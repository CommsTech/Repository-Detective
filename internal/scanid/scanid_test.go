package scanid_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/scanid"
)

func TestScanIDRoundTrip(t *testing.T) {
	id := scanid.New()
	if id == "" || id == "unknown" {
		t.Fatalf("unexpected scan id: %q", id)
	}

	ctx := scanid.With(context.Background(), id)
	if got := scanid.From(ctx); got != id {
		t.Fatalf("expected %q, got %q", id, got)
	}
}
