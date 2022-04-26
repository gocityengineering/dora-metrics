package dorametrics

import (
	"fmt"
	"strings"

	au "github.com/logrusorgru/aurora"
)

func describeDeployment(deployment DeploymentInfo) string {
	var items []string
	items = append(items, fmt.Sprintf("%s:", au.Bold(au.Cyan("DEBUG"))))
	items = append(items, fmt.Sprintf("Name=%s", au.Bold(deployment.Name)))
	items = append(items, fmt.Sprintf(" Namespace=%s", au.Bold(deployment.Namespace)))
	items = append(items, fmt.Sprintf(" Replicas=%d", au.Bold(deployment.Replicas)))
	items = append(items, fmt.Sprintf(" ReadyReplicas=%d", au.Bold(deployment.ReadyReplicas)))
	items = append(items, fmt.Sprintf(" ErrorStart=%d", au.Bold(deployment.ErrorStart)))
	return strings.Join(items, " ")
}
