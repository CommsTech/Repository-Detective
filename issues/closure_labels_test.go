package issues

import "testing"

func TestClosureLifecycleLabels(t *testing.T) {
	labels := ExpandLifecycleLabels(LifecycleResolvedVerified)
	if len(labels) != 1 || labels[0] != "repository-detective/resolved-verified" {
		t.Fatalf("unexpected labels %v", labels)
	}

	labels = ExpandLifecycleLabels(LifecycleFixPRMerged)
	if len(labels) != 1 || labels[0] != "repository-detective/fix-pr-merged" {
		t.Fatalf("unexpected labels %v", labels)
	}

	labels = ExpandLifecycleLabels(LifecyclePendingRescan)
	if len(labels) != 1 || labels[0] != "repository-detective/pending-rescan" {
		t.Fatalf("unexpected labels %v", labels)
	}
}
