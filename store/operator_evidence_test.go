package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestOperatorEvidenceWebhookAndFirstScan(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: t.TempDir() + "/ev.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.RecordWebhookDelivery(ctx, store.WebhookDeliveryEvidence{
		EventKind:  "push",
		Repository: "acme/demo",
		CommitSHA:  "abc123",
	}); err != nil {
		t.Fatal(err)
	}
	ev, ok, err := s.GetWebhookDeliveryEvidence(ctx)
	if err != nil || !ok || ev.Repository != "acme/demo" {
		t.Fatalf("webhook evidence: ok=%v err=%v %+v", ok, err, ev)
	}
	if err := s.RecordFirstScanProven(ctx, store.FirstScanEvidence{
		ScanID: "scan-1", RepositoryID: 1, TriggerType: store.TriggerPush,
		Status: "completed", RequiredOK: 2, RequiredTotal: 2, ViaWebhook: true,
	}); err != nil {
		t.Fatal(err)
	}
	fs, ok, err := s.GetFirstScanEvidence(ctx)
	if err != nil || !ok || fs.ScanID != "scan-1" || !fs.ViaWebhook {
		t.Fatalf("first scan: ok=%v err=%v %+v", ok, err, fs)
	}
}
