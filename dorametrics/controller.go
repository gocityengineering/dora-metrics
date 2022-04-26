package dorametrics

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"

	"time"

	au "github.com/logrusorgru/aurora"
	"github.com/prometheus/client_golang/prometheus"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const annotationPrefix = "dora-controller"
const annotationNameReportBefore = "report-before"
const annotationNameCycleTime = "cycle-time"
const annotationNameSuccess = "success"
const maxCycleTimeSeconds = 7200
const maxTimeToRecoverySeconds = 7200

// NewController constructs the central controller state
func NewController(
	queue workqueue.RateLimitingInterface,
	indexer cache.Indexer,
	informer cache.Controller,
	clientset kubernetes.Interface,
	mutex *sync.Mutex,
	state map[string]DeploymentInfo,
	dedup map[string]string,
	debug bool,
	collectors *Collectors) *Controller {
	return &Controller{
		Informer:   informer,
		Indexer:    indexer,
		Queue:      queue,
		Clientset:  clientset,
		Mutex:      mutex,
		State:      state,
		Dedup:      dedup,
		Debug:      debug,
		Collectors: collectors,
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)

	err := c.syncToStdout(key.(string))
	c.handleErr(err, key)

	return true
}

func (c *Controller) syncToStdout(key string) error {
	obj, keyExists, err := c.Indexer.GetByKey(key)
	if err != nil {
		log.Println(fmt.Sprintf("%s: fetching object with key %s from store failed with %v",
			au.Bold(au.Red("Error")),
			key,
			err))
		return err
	}

	// exit condition 1: nil interface received (e.g. after manual resource deletion)
	// nothing to do here; return gracefully
	if obj == nil {
		return nil
	}

	name := obj.(*appsv1.Deployment).GetName()
	namespace := obj.(*appsv1.Deployment).ObjectMeta.Namespace
	replicas := obj.(*appsv1.Deployment).Spec.Replicas             // *int32
	readyReplicas := obj.(*appsv1.Deployment).Status.ReadyReplicas // int32

	// exit condition 2: deployment has been deleted
	if !keyExists {
		log.Println(fmt.Sprintf("%s: deployment %s in namespace %s has been deleted",
			au.Bold(au.Cyan("INFO")),
			name,
			namespace))
		return nil
	}

	// create single-string lookup key; we'll use it more than once
	lookupKey := namespace + name

	if c.Debug {
		log.Println(fmt.Sprintf("%s: processing deployment %s", au.Bold(au.Cyan("INFO")), au.Bold(name)))
	}

	// keep in mind annotation values are all strings, even '100' and 'true'
	reportBeforeAnnotation := obj.(*appsv1.Deployment).ObjectMeta.Annotations[fmt.Sprintf("%s/%s", annotationPrefix, annotationNameReportBefore)]
	cycleTimeAnnotation := obj.(*appsv1.Deployment).ObjectMeta.Annotations[fmt.Sprintf("%s/%s", annotationPrefix, annotationNameCycleTime)]
	successAnnotation := obj.(*appsv1.Deployment).ObjectMeta.Annotations[fmt.Sprintf("%s/%s", annotationPrefix, annotationNameSuccess)]

	// deduplication: ignore annotations if we've already seen this update
	processAnnotations := true
	if _, ok := c.Dedup[lookupKey]; ok {
		if c.Dedup[lookupKey] == reportBeforeAnnotation {
			processAnnotations = false
		}
	}
	c.Dedup[lookupKey] = reportBeforeAnnotation

	now := time.Now()
	unixTimeSeconds := now.Unix()

	if processAnnotations {
		// we don't measure lead time for failed deployments
		reportBeforeSeconds, err := strconv.Atoi(reportBeforeAnnotation)
		if err != nil {
			log.Println(fmt.Sprintf(
				"%s: cannot parse annotation %s/%s=%s: %v",
				au.Bold(au.Red("Error")),
				annotationPrefix,
				annotationNameReportBefore,
				reportBeforeAnnotation,
				err))
			return err
		}
		if int64(reportBeforeSeconds) > unixTimeSeconds {
			if successAnnotation == "true" {
				// cycle time must be a positive integer
				cycleTimeSeconds, err := strconv.Atoi(cycleTimeAnnotation)
				if err == nil && cycleTimeSeconds > 0 {
					if cycleTimeSeconds > maxCycleTimeSeconds {
						cycleTimeSeconds = maxCycleTimeSeconds
					}
					log.Println(fmt.Sprintf("%s: submitting cycle time %d for deployment %s in namespace %s", au.Bold(au.Cyan("INFO")), au.Bold(cycleTimeSeconds), au.Bold(name), au.Bold(namespace)))
					c.Collectors.CycleTimeGauge.With(prometheus.Labels{"deployment": name, "namespace": namespace}).Set(float64(cycleTimeSeconds))
				}
				// report success
				c.Collectors.SuccessCounter.With(prometheus.Labels{"deployment": name, "namespace": namespace}).Inc()
			} else {
				log.Println(fmt.Sprintf("%s: reporting failed deployment for deployment %s in namespace %s", au.Bold(au.Cyan("INFO")), au.Bold(name), au.Bold(namespace)))
				c.Collectors.FailureCounter.With(prometheus.Labels{"deployment": name, "namespace": namespace}).Inc()
			}
		} else {
			if c.Debug {
				log.Println(fmt.Sprintf("%s: ignoring stale annotations for deployment %s in namespace %s", au.Bold(au.Cyan("INFO")), au.Bold(name), au.Bold(namespace)))
			}
		}
	}

	if _, ok := c.State[lookupKey]; !ok {
		c.State[lookupKey] = DeploymentInfo{
			name,
			namespace,
			*replicas,
			readyReplicas,
			0, // flag no error on creation
		}
	}

	var errorStart int64
	errorStart = 0

	if (*replicas) > 0 && readyReplicas == 0 {
		// failed state
		// set errorStart unless already set
		if c.State[lookupKey].ErrorStart == 0 {
			log.Println(fmt.Sprintf("%s: entered error state for deployment %s in namespace %s", au.Bold(au.Cyan("INFO")), au.Bold(name), au.Bold(namespace)))

			errorStart = unixTimeSeconds
			info := c.State[lookupKey]
			info.ErrorStart = errorStart
			c.State[lookupKey] = info
			c.Collectors.DowntimeCounter.With(prometheus.Labels{"deployment": name, "namespace": namespace}).Inc()
		}
	} else if (*replicas) == readyReplicas {
		// non-failed state (may still be an impaired deployment)
		// we ignore these for the purposes of DORA reporting
		// set TTR if ErrorStart > 0
		// then reset errorStart to 0
		if c.State[lookupKey].ErrorStart > 0 {
			timeToRecovery := unixTimeSeconds - c.State[lookupKey].ErrorStart
			if timeToRecovery > maxTimeToRecoverySeconds {
				timeToRecovery = maxTimeToRecoverySeconds
			}
			log.Println(fmt.Sprintf("%s: left error state for deployment %s in namespace %s: TTR was %d", au.Bold(au.Cyan("INFO")), au.Bold(name), au.Bold(namespace), au.Bold(timeToRecovery)))
			c.Collectors.TimeToRecoveryGauge.With(prometheus.Labels{"deployment": name, "namespace": namespace}).Set(math.Round(float64(timeToRecovery)))
			info := c.State[lookupKey]
			info.ErrorStart = 0
			c.State[lookupKey] = info
		}
	}

	deployment := DeploymentInfo{
		name,
		namespace,
		*replicas,
		readyReplicas,
		errorStart,
	}

	if c.Debug {
		log.Println(describeDeployment(deployment))
	}

	if c.Debug {
		bytes, err := json.Marshal(deployment)
		if err != nil {
			log.Println(fmt.Sprintf("%s: %s", au.Bold(au.Red("Error")), au.Bold(err)))
			return nil
		}
		// main JSON output goes to stdout
		fmt.Printf("%s\n", bytes)
	}

	if c.Queue.Len() == 0 {
		// ignore: not significant for now
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.Queue.Forget(key)
		return
	}

	if c.Queue.NumRequeues(key) < 5 {
		log.Println(fmt.Sprintf("%s: can't sync deployment %v: %v", au.Bold(au.Red("Error")), key, err))
		c.Queue.AddRateLimited(key)
		return
	}

	c.Queue.Forget(key)
	runtime.HandleError(err)
	log.Println(fmt.Sprintf("%s: dropping deployment %q from the queue: %v", au.Bold(au.Cyan("INFO")), key, err))
}

// Run manages the controller lifecycle
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	defer c.Queue.ShutDown()
	log.Println(fmt.Sprintf("%s: starting DORA controller", au.Bold(au.Cyan("INFO"))))

	go c.Informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.Informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Println(fmt.Sprintf("%s: stopping DORA controller", au.Bold(au.Cyan("INFO"))))
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
