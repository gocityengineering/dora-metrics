package dorametrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterCollectors(collectors *Collectors, dryrun bool) error {
	collectors.SuccessCounter = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dora_successful_deployments_total",
		Help: "counter for successful deployments",
	},
		[]string{
			"deployment",
			"namespace",
		})
	if !dryrun {
		prometheus.MustRegister(collectors.SuccessCounter)
	}

	collectors.FailureCounter = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dora_failed_deployments_total",
		Help: "counter for failed deployments",
	},
		[]string{
			"deployment",
			"namespace",
		})

	if !dryrun {
		prometheus.MustRegister(collectors.FailureCounter)
	}

	collectors.DowntimeCounter = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dora_downtime_total",
		Help: "counter for periods of downtime",
	},
		[]string{
			"deployment",
			"namespace",
		})

	if !dryrun {
		prometheus.MustRegister(collectors.DowntimeCounter)
	}

	collectors.TimeToRecoveryGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dora_time_to_recovery_seconds",
		Help: "gauge for time to recovery",
	},
		[]string{
			"deployment",
			"namespace",
		})

	if !dryrun {
		prometheus.MustRegister(collectors.TimeToRecoveryGauge)
	}

	collectors.CycleTimeGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dora_cycle_time_seconds",
		Help: "gauge for cycle time",
	},
		[]string{
			"deployment",
			"namespace",
		})

	if !dryrun {
		prometheus.MustRegister(collectors.CycleTimeGauge)
	}

	return nil
}
