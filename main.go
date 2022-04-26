package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	dorametrics "github.com/gocityengineering/dora-metrics/dorametrics"
	au "github.com/logrusorgru/aurora"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

const labelPrefix = "dora-controller"
const labelNameEnabled = "enabled"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(0)
	}

	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	master := flag.String("master", "", "master url")
	debug := flag.Bool("debug", false, "debug mode")

	flag.Parse()

	os.Exit(realMain(*kubeconfig, *master, *debug, false))
}

func realMain(kubeconfig, master string, debug, dryrun bool) int {
	// register collectors
	var collectors = dorametrics.Collectors{}
	err := dorametrics.RegisterCollectors(&collectors, dryrun)
	if err != nil {
		fmt.Fprintf(os.Stderr, `Can't register collectors: %v`, err)
		return 1
	}

	// API requests will fail unless we set InsecureSkipVerify to true
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// set up controller

	// support out-of-cluster deployments (param, env var only)
	if len(kubeconfig) == 0 {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	var config *rest.Config
	var configError error

	if len(kubeconfig) > 0 {
		config, configError = clientcmd.BuildConfigFromFlags(master, kubeconfig)
		if configError != nil {
			fmt.Fprintf(os.Stderr, "%s: %s", au.Bold(au.Red("Out-of-cluster error")), configError)
			return 2
		}
	} else {
		config, configError = rest.InClusterConfig()
		if configError != nil {
			fmt.Fprintf(os.Stderr, "%s: %s", au.Bold(au.Red("In-cluster error")), configError)
			return 3
		}
	}

	// create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", au.Bold(au.Red("Error")), err)
		return 4
	}

	var mutex = &sync.Mutex{}
	var state = map[string]dorametrics.DeploymentInfo{}

	labelKey := fmt.Sprintf("%s/%s", labelPrefix, labelNameEnabled)
	labelValue := "true"
	deploymentSelector := labels.SelectorFromSet(labels.Set(map[string]string{labelKey: labelValue})).String()
	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = deploymentSelector
	}
	deploymentListWatcher := cache.NewFilteredListWatchFromClient(
		clientset.AppsV1().RESTClient(),
		"deployments",
		metav1.NamespaceAll,
		optionsModifier)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(deploymentListWatcher, &appsv1.Deployment{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	dedup := make(map[string]string)
	controller := dorametrics.NewController(
		queue,
		indexer,
		informer,
		clientset,
		mutex,
		state,
		dedup,
		debug,
		&collectors)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

	return 0
}
