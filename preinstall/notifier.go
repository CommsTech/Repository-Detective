package preinstall

import "git.commsnet.org/commstech/repository-detective/store"

// AuditNotifier receives sanitized pre-install audit events (optional).
type AuditNotifier interface {
	OnAuditComplete(req store.AuditRequest, findingCount int)
	OnDisclosureReport(auditID, reportType string)
}
