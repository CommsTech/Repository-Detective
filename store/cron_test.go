package store_test

import (
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestValidateCronExpression(t *testing.T) {
	valid := []string{"0 2 * * *", "0 */6 * * *", "30 3 * * 1", "*/15 * * * *"}
	for _, expr := range valid {
		if err := store.ValidateCronExpression(expr); err != nil {
			t.Fatalf("expected valid cron %q: %v", expr, err)
		}
	}
	for _, expr := range []string{"", "not cron", "0 2 * *"} {
		if err := store.ValidateCronExpression(expr); err == nil {
			t.Fatalf("expected invalid cron %q", expr)
		}
	}
}

func TestNextCronRun(t *testing.T) {
	base := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	next, err := store.NextCronRun("0 2 * * *", base)
	if err != nil {
		t.Fatal(err)
	}
	if !next.After(base) {
		t.Fatalf("next should be after base: %v", next)
	}
	if next.Hour() != 2 || next.Minute() != 0 {
		t.Fatalf("unexpected next run: %v", next)
	}
}

func TestDescribeCron(t *testing.T) {
	desc := store.DescribeCron("0 2 * * *", time.Now().UTC())
	if !desc.Valid || desc.NextRun == "" {
		t.Fatalf("unexpected describe: %+v", desc)
	}
	bad := store.DescribeCron("bad cron", time.Now().UTC())
	if bad.Valid {
		t.Fatal("expected invalid cron description")
	}
}
