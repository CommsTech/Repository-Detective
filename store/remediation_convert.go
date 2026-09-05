package store

import (
	"encoding/json"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

// RemediationPlanFromDomain converts a domain plan to a store record.
func RemediationPlanFromDomain(plan remediation.Plan) RemediationPlanRecord {
	var findingID *int64
	if plan.FindingID > 0 {
		v := plan.FindingID
		findingID = &v
	}
	var repositoryID *int64
	if plan.RepositoryID > 0 {
		v := plan.RepositoryID
		repositoryID = &v
	}
	var auditID *string
	if plan.AuditID != "" {
		v := plan.AuditID
		auditID = &v
	}
	affected, _ := json.Marshal(plan.AffectedFiles)
	tests, _ := json.Marshal(plan.RequiredTests)
	commands, _ := json.Marshal(plan.ValidationCommands)
	blocked, _ := json.Marshal(plan.BlockedReasons)
	return RemediationPlanRecord{
		PlanID:                 plan.ID,
		FindingID:              findingID,
		RepositoryID:           repositoryID,
		AuditID:                auditID,
		Fingerprint:            plan.Fingerprint,
		Category:               plan.Category,
		Severity:               plan.Severity,
		Source:                 plan.Source,
		RuleID:                 plan.RuleID,
		Title:                  plan.Title,
		Summary:                plan.Summary,
		FixStrategy:            plan.FixStrategy,
		AffectedFilesJSON:      affected,
		RequiredTestsJSON:      tests,
		ValidationCommandsJSON: commands,
		RegressionRisk:         plan.RegressionRisk,
		FixComplexity:          plan.FixComplexity,
		SafeForAutoPR:          plan.SafeForAutoPR,
		RequiresHumanReview:    plan.RequiresHumanReview,
		BlockedReasonsJSON:     blocked,
		Advisory:                 plan.Advisory,
		Status:                 plan.Status,
		CreatedAt:              plan.CreatedAt,
		UpdatedAt:              plan.UpdatedAt,
	}
}

// RemediationPlanToDomain converts a store record to a domain plan.
func RemediationPlanToDomain(rec RemediationPlanRecord) remediation.Plan {
	var affected, tests, commands, blocked []string
	_ = json.Unmarshal(rec.AffectedFilesJSON, &affected)
	_ = json.Unmarshal(rec.RequiredTestsJSON, &tests)
	_ = json.Unmarshal(rec.ValidationCommandsJSON, &commands)
	_ = json.Unmarshal(rec.BlockedReasonsJSON, &blocked)
	plan := remediation.Plan{
		ID:                  rec.PlanID,
		Fingerprint:         rec.Fingerprint,
		Category:            rec.Category,
		Severity:            rec.Severity,
		Source:              rec.Source,
		RuleID:              rec.RuleID,
		Title:               rec.Title,
		Summary:             rec.Summary,
		FixStrategy:         rec.FixStrategy,
		AffectedFiles:       affected,
		RequiredTests:       tests,
		ValidationCommands:  commands,
		RegressionRisk:      rec.RegressionRisk,
		FixComplexity:       rec.FixComplexity,
		SafeForAutoPR:       rec.SafeForAutoPR,
		RequiresHumanReview: rec.RequiresHumanReview,
		BlockedReasons:      blocked,
		Advisory:            rec.Advisory,
		Status:              rec.Status,
		CreatedAt:           rec.CreatedAt,
		UpdatedAt:           rec.UpdatedAt,
	}
	if rec.FindingID != nil {
		plan.FindingID = *rec.FindingID
	}
	if rec.RepositoryID != nil {
		plan.RepositoryID = *rec.RepositoryID
	}
	if rec.AuditID != nil {
		plan.AuditID = *rec.AuditID
	}
	return plan
}
