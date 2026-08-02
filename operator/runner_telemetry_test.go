package operator_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/operator"
)

func TestToolStatusStates(t *testing.T) {
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	tests := []struct {
		name      string
		cfg       operator.ScannerConfig
		tool      string
		wantState string
	}{
		{
			name:      "disabled by config",
			cfg:       operator.ScannerConfig{},
			tool:      "checkov",
			wantState: operator.StatusDisabledByConfig,
		},
		{
			name:      "enabled missing binary",
			cfg:       operator.ScannerConfig{EnableGosec: true},
			tool:      "gosec",
			wantState: operator.StatusEnabledMissingBinary,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tools := operator.CheckTools(tc.cfg)
			for _, tool := range tools {
				if tool.Name == tc.tool {
					if tool.StatusState != tc.wantState {
						t.Fatalf("got %s want %s action=%q", tool.StatusState, tc.wantState, tool.Action)
					}
					if tool.Action == "" && tc.wantState != operator.StatusEnabledAvailable {
						t.Fatal("expected action guidance")
					}
					return
				}
			}
			t.Fatalf("tool %s not found", tc.tool)
		})
	}
}

func TestRunnerTelemetryDisabledGlobal(t *testing.T) {
	view := operator.BuildRunnerTelemetry(false, nil, nil, "")
	if view.State != "disabled_global" {
		t.Fatalf("got %s", view.State)
	}
	if !stringsContains(view.Message, "disabled") {
		t.Fatalf("message: %s", view.Message)
	}
}

func TestRunnerTelemetryEnabledNoJobs(t *testing.T) {
	view := operator.BuildRunnerTelemetry(true, map[string]int{}, nil, "")
	if view.State != "enabled_no_jobs" {
		t.Fatalf("got %s", view.State)
	}
}

func TestRunnerTelemetryHasJobs(t *testing.T) {
	view := operator.BuildRunnerTelemetry(true, map[string]int{"completed": 2, "failed": 1}, nil, "")
	if view.State != "has_jobs" || view.TotalJobs != 3 {
		t.Fatalf("got %+v", view)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
