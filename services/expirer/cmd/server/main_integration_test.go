//go:build integration

package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	service "github.com/LgAcerbi/go-video-upload/services/expirer/internal/application/services"
)

type expirerRunnerFake struct {
	mu        sync.Mutex
	calls     int
	returnErr error
}

func (f *expirerRunnerFake) ExpireStaleUploads(context.Context, int) (service.ExpireResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.returnErr != nil {
		return service.ExpireResult{}, f.returnErr
	}
	return service.ExpireResult{Found: 2, Expired: 1, Skipped: 1}, nil
}

func TestRunScheduler_ImmediateRunOnStart_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := &expirerRunnerFake{}
	log := logger.New(&logger.Config{Service: "expirer-it"})
	runScheduler(ctx, log, runner, 10, 200*time.Millisecond, time.Hour)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runner.mu.Lock()
		calls := runner.calls
		runner.mu.Unlock()
		if calls >= 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected immediate run on start")
}

func TestRunScheduler_PeriodicRuns_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := &expirerRunnerFake{}
	log := logger.New(&logger.Config{Service: "expirer-it"})
	runScheduler(ctx, log, runner, 10, 300*time.Millisecond, 120*time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runner.mu.Lock()
		calls := runner.calls
		runner.mu.Unlock()
		if calls >= 3 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected periodic runs to execute")
}

func TestRunScheduler_ContinuesAfterError_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := &expirerRunnerFake{returnErr: errors.New("upstream unavailable")}
	log := logger.New(&logger.Config{Service: "expirer-it"})
	runScheduler(ctx, log, runner, 10, 300*time.Millisecond, 120*time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runner.mu.Lock()
		calls := runner.calls
		runner.mu.Unlock()
		if calls >= 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected scheduler to continue invoking after errors")
}
