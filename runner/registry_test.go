package runner_test

import (
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestRegistryRecordAndListHeartbeats(t *testing.T) {
	reg := runner.NewRegistry()
	reg.RecordHeartbeat(runner.WorkerHeartbeat{RunnerID: "worker-a", Version: "1.0"})
	reg.RecordHeartbeat(runner.WorkerHeartbeat{RunnerID: "worker-b", Version: "1.0", LastSeenAt: time.Now().UTC().Add(-1 * time.Hour)})

	workers := reg.ListHeartbeats(15 * time.Minute)
	if len(workers) != 1 {
		t.Fatalf("expected 1 recent worker, got %d", len(workers))
	}
	if workers[0].RunnerID != "worker-a" {
		t.Fatalf("got %q", workers[0].RunnerID)
	}
}

func TestRegistryIgnoresEmptyRunnerID(t *testing.T) {
	reg := runner.NewRegistry()
	reg.RecordHeartbeat(runner.WorkerHeartbeat{RunnerID: ""})
	if len(reg.ListHeartbeats(time.Hour)) != 0 {
		t.Fatal("expected empty list for blank runner id")
	}
}
