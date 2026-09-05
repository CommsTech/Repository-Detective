package learning

import (
	"context"
	"encoding/json"
	"strconv"

	"git.commsnet.org/commstech/repository-detective/store"
)

// EventRecorder persists learning events.
type EventRecorder interface {
	RecordLearningEvent(ctx context.Context, ev store.LearningEvent) (store.LearningEvent, error)
}

// Emit records a learning event with idempotency.
func Emit(ctx context.Context, rec EventRecorder, ev store.LearningEvent) error {
	if rec == nil || ev.RepositoryID <= 0 || ev.EventType == "" {
		return nil
	}
	if ev.IdempotencyKey == "" {
		ev.IdempotencyKey = defaultIdempotencyKey(ev)
	}
	_, err := rec.RecordLearningEvent(ctx, ev)
	return err
}

func defaultIdempotencyKey(ev store.LearningEvent) string {
	fid := int64(0)
	if ev.FindingID != nil {
		fid = *ev.FindingID
	}
	return ev.EventType + ":" + ev.ScanID + ":" + ev.Fingerprint + ":" + strconv.FormatInt(fid, 10)
}

// EmitJSON attaches structured evidence.
func EmitJSON(ctx context.Context, rec EventRecorder, ev store.LearningEvent, evidence any) error {
	if evidence != nil {
		raw, err := json.Marshal(evidence)
		if err == nil {
			ev.EvidenceJSON = raw
		}
	}
	return Emit(ctx, rec, ev)
}
