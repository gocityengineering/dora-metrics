package dorametrics

import (
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/prometheus/client_golang/prometheus"
)

type ControllerConfig struct {
	Targets []Target `json:"deployments"`
	Stage   string   `json:"stage"`
}

type Target struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Team      string `json:"team"`
	Alert     bool
}

// Controller represents the controller state
type Controller struct {
	Indexer    cache.Indexer
	Queue      workqueue.RateLimitingInterface
	Informer   cache.Controller
	Clientset  kubernetes.Interface
	Mutex      *sync.Mutex
	State      map[string]DeploymentInfo // map[NAMESPACE:NAME]DeploymentInfo
	Dedup      map[string]string         // map[NAMESPACE:NAME]REPORT_BEFORE
	Debug      bool
	Collectors *Collectors
}

// DeploymentInfo captures the information written to stdout
type DeploymentInfo struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Replicas      int32  `json:"replicas"`
	ReadyReplicas int32  `json:"readyReplicas"`
	ErrorStart    int64  `json:"errorStart"`
}

type Collectors struct {
	CycleTimeGauge      prometheus.GaugeVec
	TimeToRecoveryGauge prometheus.GaugeVec
	SuccessCounter      prometheus.CounterVec
	FailureCounter      prometheus.CounterVec
	DowntimeCounter     prometheus.CounterVec
}
