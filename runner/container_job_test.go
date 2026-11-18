package runner_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/runner"
)

func TestExecuteContainerImageScanMissingPayload(t *testing.T) {
	t.Parallel()
	result, err := runner.ExecuteJob(context.Background(), runner.JobSpec{
		JobID: "j1", ScanID: "s1", JobType: runner.JobTypeContainerImageScan,
	}, runner.JobExecuteInput{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != runner.JobStatusFailed {
		t.Fatalf("status %q", result.Status)
	}
}

func TestJobTypeAllowedContainerScan(t *testing.T) {
	t.Parallel()
	if !runner.JobTypeAllowed([]string{runner.JobTypeContainerImageScan}, runner.JobTypeContainerImageScan) {
		t.Fatal("container scan should be allowed")
	}
}
