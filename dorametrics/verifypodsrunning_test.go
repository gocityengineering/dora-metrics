package dorametrics

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func podPhase(phase string) v1.PodPhase {
	switch phase {
	case "Running":
		return v1.PodRunning
	case "Pending":
		return v1.PodPending
	case "Unknown":
		return v1.PodUnknown
	}
	return v1.PodUnknown
}

func pod(phase string) *v1.Pod {
	image := "ubuntu"
	return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "server-a", Namespace: "default"}, Spec: v1.PodSpec{Containers: []v1.Container{{Image: image}}}, Status: v1.PodStatus{Phase: podPhase(phase)}}
}

func TestVerifyPodsRunning(t *testing.T) {
	var tests = []struct {
		description string
		result      bool
		objs        []runtime.Object
	}{
		{"pod_running", true, []runtime.Object{pod("Running")}},
		{"pod_pending", false, []runtime.Object{pod("Pending")}},
		{"pod_unknown", false, []runtime.Object{pod("Unknown")}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			client := fake.NewSimpleClientset(test.objs...)
			result := verifyPodsRunning(client, "default", "server-a", "ubuntu")
			expected := "success"
			if test.result == false {
				expected = "failure"
			}

			if result != test.result {
				t.Errorf("Expected function to return %s", expected)
			}
		})
	}
}
