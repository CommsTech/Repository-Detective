package ui

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
)

// CalibrationRecommendationView decorates a pending recommendation for the Learning UI.
type CalibrationRecommendationView struct {
	store.CalibrationRecommendation
	CanAccept   bool
	AcceptLabel string
	AcceptHint  string
	BlockReason string
}

func enrichCalibrationRecommendationViews(recs []store.CalibrationRecommendation) []CalibrationRecommendationView {
	out := make([]CalibrationRecommendationView, 0, len(recs))
	for _, rec := range recs {
		view := CalibrationRecommendationView{CalibrationRecommendation: rec}
		if learning.IsProtectedFromAutoDowngrade("", rec.Category) {
			view.BlockReason = "Protected security/secret category — mark findings false-positive individually instead of accepting a rule suppression."
			out = append(out, view)
			continue
		}
		view.CanAccept = true
		if strings.EqualFold(rec.Scope, "global") {
			view.AcceptLabel = "Accept for affected repos"
			view.AcceptHint = "Creates repo-scoped report_only rules for repositories that already have this finding — findings stay visible; high/critical are never downgraded."
		} else {
			view.AcceptLabel = "Accept"
			view.AcceptHint = "Creates a repo-scoped report_only calibration rule — findings stay visible; high/critical are never downgraded."
		}
		out = append(out, view)
	}
	return out
}
