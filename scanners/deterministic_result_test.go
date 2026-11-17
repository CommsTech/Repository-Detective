package scanners_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestDeterministicRunResult(t *testing.T) {
	clean := scanners.DeterministicRunResult("health", 0)
	if clean.Scanner != "health" || clean.Status != scanners.StatusClean {
		t.Fatalf("unexpected clean result: %+v", clean)
	}
	found := scanners.DeterministicRunResult("static", 3)
	if found.Status != scanners.StatusFound {
		t.Fatalf("expected found status, got %s", found.Status)
	}
}
