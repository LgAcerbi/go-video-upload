package integration

import (
	"testing"
	"time"
)

func Eventually(t *testing.T, timeout, interval time.Duration, assertion func() bool, failureMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if assertion() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("eventually timeout: %s", failureMsg)
		}
		time.Sleep(interval)
	}
}
