package main

import (
	"os"
	"testing"
)

func TestDefaultHostDetails(t *testing.T) {
	// XXX: This is lazy since it will pass on aws (which we may run these tests
	// on someday)
	hostType := GetHostType("http://169.254.169.254")
	hostDetails, err := hostType.Details()

	if err != nil {
		t.Fatal(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	if hostDetails.Hostname != hostname {
		t.Fatalf("Unexpected hostname (%s) %v ", hostDetails.Hostname, err)
	}
}
