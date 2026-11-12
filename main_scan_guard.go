package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

// checkScanAdmission returns whether a new scan should be refused for cooldown or overlap.
// Push and pull-request webhooks are exempt so legitimate forge events are not delayed.
func checkScanAdmission(ctx context.Context, forgeType, owner, repo, triggerType string) (bool, string) {
	if rdStore == nil {
		return false, ""
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return false, ""
	}
	if triggerType == store.TriggerPush || triggerType == store.TriggerPR {
		return false, ""
	}

	fullName := owner + "/" + repo
	dbRepo, err := rdStore.GetRepositoryByFullName(ctx, forgeType, fullName)
	if err != nil {
		return false, ""
	}

	running, err := rdStore.HasRunningScanForRepository(ctx, dbRepo.ID)
	if err != nil {
		logger.Warnf("Scan admission: running check failed for %s: %v", fullName, err)
		return false, ""
	}
	if running {
		return true, "repository already has a running scan"
	}

	cooldownSec := config.ScanCooldownSeconds
	if cooldownSec <= 0 {
		return false, ""
	}
	lastStarted, err := rdStore.GetLastScanStartedAt(ctx, dbRepo.ID)
	if err != nil {
		logger.Warnf("Scan admission: last scan lookup failed for %s: %v", fullName, err)
		return false, ""
	}
	if lastStarted == nil {
		return false, ""
	}
	elapsed := time.Since(*lastStarted)
	cooldown := time.Duration(cooldownSec) * time.Second
	if elapsed < cooldown {
		remaining := cooldown - elapsed
		return true, fmt.Sprintf("scan cooldown active (%s since last scan, retry after %s)",
			elapsed.Round(time.Second), remaining.Round(time.Second))
	}
	return false, ""
}
