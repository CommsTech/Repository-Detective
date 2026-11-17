package runner_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestNativeRunnerClaimsRespectCapabilities(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "claim.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "project", FullName: "team/project"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, typ := range []string{runner.JobTypeRemediationVerify, runner.JobTypeGraph} {
		id := "rj-" + typ
		spec, err := json.Marshal(runner.BuildJobSpecForType(runner.Config{}, id, typ, repo, "", "main", "", analyzers.PolicySnapshot{}, nil))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRunnerJob(ctx, store.RunnerJob{JobID: id, RepositoryID: repo.ID, JobType: typ, Status: store.RunnerJobStatusQueued, RunnerMode: runner.ModeNative, JobSpecJSON: spec}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ClaimNextRunnerJobForTypes(ctx, now, []string{runner.JobTypeGraph})
	if err != nil || got.JobType != runner.JobTypeGraph {
		t.Fatalf("graph worker claimed %v err %v", got.JobType, err)
	}
	other, err := s.GetRunnerJob(ctx, "rj-"+runner.JobTypeRemediationVerify)
	if err != nil || other.Status != store.RunnerJobStatusQueued {
		t.Fatalf("verify job was stolen: %+v err %v", other, err)
	}
	recv := runner.NewReceiver(s, runner.Config{}, nil, nil)
	got2, spec, err := recv.ClaimNextJobForTypes(ctx, []string{runner.JobTypeRemediationVerify})
	if err != nil || got2.JobType != runner.JobTypeRemediationVerify || spec.JobID != got2.JobID {
		t.Fatalf("verification worker claim %v %+v err %v", got2.JobType, spec, err)
	}
	if _, _, err := recv.ClaimNextJobForTypes(ctx, nil); err == nil {
		t.Fatal("empty capabilities must not claim jobs")
	}
}

func TestRunnerRegistryBindsAdvertisedJobTypes(t *testing.T) {
	reg := runner.NewRegistry()
	reg.RecordHeartbeat(runner.WorkerHeartbeat{RunnerID: "a", Capabilities: []string{runner.JobTypeGraph}, LastSeenAt: time.Now().UTC()})
	if !reg.MatchesCapabilities("a", []string{runner.JobTypeGraph}, time.Minute) {
		t.Fatal("matching heartbeat rejected")
	}
	if reg.MatchesCapabilities("a", []string{runner.JobTypeRemediationVerify}, time.Minute) {
		t.Fatal("claim exceeded worker capabilities")
	}
	if reg.MatchesCapabilities("other", []string{runner.JobTypeGraph}, time.Minute) {
		t.Fatal("unknown worker matched")
	}
}
