package dorametrics

import "strings"

func getDeploymentFlags(success bool, previousSuccess bool, imageChanged bool) string {
	var flags []string

	// overall outcome
	if success {
		flags = append(flags, "DORA_SUCCESS")
	} else {
		flags = append(flags, "DORA_FAILURE")
	}

	// deployment outcome
	if imageChanged {
		flags = append(flags, "DORA_NEW_IMAGE")
		if success {
			flags = append(flags, "DORA_SUCCESSFUL_DEPLOYMENT")
		} else {
			flags = append(flags, "DORA_FAILED_DEPLOYMENT")
		}
	} else {
		flags = append(flags, "DORA_SAME_IMAGE")
	}

	// after success/failure
	if previousSuccess {
		flags = append(flags, "DORA_PREVIOUS_SUCCESS")
	} else {
		flags = append(flags, "DORA_PREVIOUS_FAILURE")
		if success {
			flags = append(flags, "DORA_RECOVERY")
		} else {
			flags = append(flags, "DORA_REPEAT_FAILURE")
		}
	}

	return strings.Join(flags, "|")
}
