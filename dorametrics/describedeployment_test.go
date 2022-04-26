package dorametrics

import (
	"testing"

	"github.com/acarl005/stripansi"
)

func TestDescribeDeployment(t *testing.T) {
	var tests = []struct {
		description string
		deployment  DeploymentInfo
		expected    string
	}{
		{"successful_deployment", DeploymentInfo{"server-c", "default", 1, 1, 0}, "DEBUG: Name=server-c  Namespace=default  Replicas=1  ReadyReplicas=1  ErrorStart=0"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := stripansi.Strip(describeDeployment(test.deployment))
			if actual != test.expected {
				t.Errorf("Unexpected description '%s'; expected '%s'", actual, test.expected)
			}
		})
	}
}
