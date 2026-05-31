package issues

import "testing"

func TestClosureLifecycleLabelsDualMode(t *testing.T) {
	SetLabelCompatMode(LabelCompatDual)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })
	labels := ExpandLifecycleLabels(LifecycleResolvedVerified)
	if len(labels) != 2 {
		t.Fatalf("expected dual labels, got %v", labels)
	}
}

func TestClosureLifecycleLabelsLegacyOnly(t *testing.T) {
	SetLabelCompatMode(LabelCompatLegacyOnly)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })
	labels := ExpandLifecycleLabels(LifecycleFixPRMerged)
	if len(labels) != 1 || labels[0] != "bugbot/fix-pr-merged" {
		t.Fatalf("unexpected labels %v", labels)
	}
}

func TestClosureLifecycleLabelsNewOnly(t *testing.T) {
	SetLabelCompatMode(LabelCompatNewOnly)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })
	labels := ExpandLifecycleLabels(LifecyclePendingRescan)
	if len(labels) != 1 || labels[0] != "repository-detective/pending-rescan" {
		t.Fatalf("unexpected labels %v", labels)
	}
}
