package runner

import "time"

// GiteaActionsBackendConfig controls optional repo-native workflow verification.
type GiteaActionsBackendConfig struct {
	Enabled                 bool
	WorkflowName            string
	TriggerMode             string
	TimeoutSeconds          int
	RequireOperatorApproval bool
}

// Normalized applies defaults.
func (c GiteaActionsBackendConfig) Normalized() GiteaActionsBackendConfig {
	out := c
	if out.WorkflowName == "" {
		out.WorkflowName = "repository-detective-verify.yml"
	}
	if out.TriggerMode == "" {
		out.TriggerMode = "workflow_dispatch"
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = 1800
	}
	return out
}

// WorkflowTriggerRequest describes a workflow_dispatch verification request.
type WorkflowTriggerRequest struct {
	Owner      string         `json:"owner"`
	Repository string         `json:"repository"`
	Ref        string         `json:"ref"`
	Workflow   string         `json:"workflow"`
	Inputs     map[string]any `json:"inputs,omitempty"`
	RequestedAt time.Time     `json:"requested_at"`
}

// BuildWorkflowTrigger builds a workflow dispatch payload for remediation verification.
func BuildWorkflowTrigger(cfg GiteaActionsBackendConfig, owner, repo, ref, scanID string) WorkflowTriggerRequest {
	cfg = cfg.Normalized()
	return WorkflowTriggerRequest{
		Owner:       owner,
		Repository:  repo,
		Ref:         ref,
		Workflow:    cfg.WorkflowName,
		RequestedAt: time.Now().UTC(),
		Inputs: map[string]any{
			"scan_id": scanID,
			"mode":    "remediation_verify",
		},
	}
}
