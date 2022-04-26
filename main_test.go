package main

import (
	"testing"
)

func TestRealMain(t *testing.T) {
	var tests = []struct {
		description string
		kubeconfig  string
		master      string
		debug       bool
		dryrun      bool
		expected    int
	}{
		{"blank kubeconfig", "", "", true, true, 3},
		{"nonblank kubeconfig", "nonesuch", "", true, true, 2},
		{"nonblank master", "", "localhost", true, true, 3},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			retVal := realMain(test.kubeconfig, test.master, test.debug, test.dryrun)
			if retVal != test.expected {
				t.Errorf("%s: unexpected return value '%d'; expected '%d'", test.description, retVal, test.expected)
			}
		})
	}
}
