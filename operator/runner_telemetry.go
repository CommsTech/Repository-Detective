package operator

import (
	"fmt"
	"strings"
	"time"
)

// RunnerTelemetryView explains runner job queue state for operators.
type RunnerTelemetryView struct {
	State          string         `json:"state"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	Detail         string         `json:"detail,omitempty"`
	Action         string         `json:"action,omitempty"`
	DocsURL        string         `json:"docs_url,omitempty"`
	JobsByStatus   map[string]int `json:"jobs_by_status,omitempty"`
	LastJobAt      *time.Time     `json:"last_job_at,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	TotalJobs      int            `json:"total_jobs"`
}

const runnerDocsURL = "/ui/docs/RUNNER_DELEGATION.md"

// BuildRunnerTelemetry produces a state-aware empty/full runner queue explanation.
func BuildRunnerTelemetry(delegationEnabled bool, jobsByStatus map[string]int, lastJobAt *time.Time, lastError string) RunnerTelemetryView {
	view := RunnerTelemetryView{
		JobsByStatus: jobsByStatus,
		LastJobAt:    lastJobAt,
		LastError:    strings.TrimSpace(lastError),
		DocsURL:      runnerDocsURL,
	}
	for _, c := range jobsByStatus {
		view.TotalJobs += c
	}

	if !delegationEnabled {
		view.State = "disabled_global"
		view.Title = "Runner delegation disabled"
		view.Message = "Runner delegation is disabled. No runner telemetry is expected."
		view.Detail = "Enable runner_delegation_enabled and configure runner_shared_secret for native Repository Detective runners. Gitea act_runner is optional for repo-native test verification — see docs/RUNNER_DELEGATION.md."
		view.Action = "See docs/RUNNER_DELEGATION.md for runner setup."
		return view
	}

	if view.TotalJobs == 0 {
		view.State = "enabled_no_jobs"
		view.Title = "No runner jobs yet"
		view.Message = "Runner delegation is enabled, but no runner jobs have been dispatched yet."
		view.Detail = "Run a scheduled or manual full scan for a repo with runner_policy=auto or gitea_actions."
		view.Action = "Set runner_policy to auto on a repository and trigger a scan."
		return view
	}

	view.State = "has_jobs"
	view.Title = "Runner jobs recorded"
	view.Message = fmt.Sprintf("Runner delegation is enabled — %d job(s) in telemetry.", view.TotalJobs)
	if view.LastJobAt != nil {
		view.Detail = "Last job: " + view.LastJobAt.UTC().Format(time.RFC3339)
	}
	return view
}

// BuildRepoRunnerTelemetry explains runner state for a single repository.
func BuildRepoRunnerTelemetry(delegationEnabled bool, runnerPolicy string, jobsByStatus map[string]int) RunnerTelemetryView {
	view := BuildRunnerTelemetry(delegationEnabled, jobsByStatus, nil, "")
	policy := strings.ToLower(strings.TrimSpace(runnerPolicy))
	if delegationEnabled && (policy == "core" || policy == "") && view.TotalJobs == 0 {
		view.State = "repo_core_only"
		view.Title = "Runs on core for this repo"
		view.Message = "Runner delegation is enabled globally, but this repo is configured to run on core."
		view.Detail = "Set runner_policy to auto or gitea_actions to create runner jobs for this repository."
		view.Action = "Update repo settings → runner policy."
	}
	return view
}
