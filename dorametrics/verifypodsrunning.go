package dorametrics

import (
	"context"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func verifyPodsRunning(clientset kubernetes.Interface, namespace string, name string, image string) bool {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})

	if err != nil {
		return false
	}

	running := 0
	for _, pod := range pods.Items {
		podName := pod.ObjectMeta.Name
		if strings.HasPrefix(podName, name) {
			for _, container := range pod.Spec.Containers {
				if container.Image == image && pod.Status.Phase == "Running" {
					running++
				} else {
					return false
				}
			}
		}
	}

	return running > 0
}
