package runner_test

import (
	"encoding/json"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestDoubleMarshalStableForHMAC(t *testing.T) {
	result := runner.JobResult{
		Version: 1, JobID: "j1", ScanID: "s1", Status: runner.JobStatusCompleted,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
		Graph: &graph.Graph{},
	}
	body1, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	body2, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(body1) != string(body2) {
		t.Fatal("expected stable JSON marshal for HMAC signing")
	}
}
