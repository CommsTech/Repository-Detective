package store

import (
	"database/sql"
	"fmt"
)

const currentSchemaVersion = 17

var migrationStatements = map[int][]string{
	1: {
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			forge_type TEXT NOT NULL DEFAULT 'gitea',
			owner TEXT NOT NULL,
			name TEXT NOT NULL,
			full_name TEXT NOT NULL,
			clone_url TEXT NOT NULL DEFAULT '',
			default_branch TEXT NOT NULL DEFAULT '',
			connected_repo INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(forge_type, full_name)
		)`,
		`CREATE TABLE IF NOT EXISTS repo_settings (
			repository_id INTEGER PRIMARY KEY,
			enabled INTEGER,
			policy_level TEXT,
			workspace_mode TEXT,
			analysis_depth INTEGER,
			enable_llm_auditors INTEGER,
			enable_trivy INTEGER,
			enable_grype INTEGER,
			enable_gitleaks INTEGER,
			enable_semgrep INTEGER,
			enable_linters INTEGER,
			severity_gate TEXT,
			confidence_gate REAL,
			issue_policy TEXT,
			remediation_policy TEXT,
			runner_policy TEXT,
			schedule_enabled INTEGER,
			schedule_cron TEXT,
			ai_policy TEXT,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS scans (
			id TEXT PRIMARY KEY,
			repository_id INTEGER NOT NULL,
			trigger_type TEXT NOT NULL,
			ref TEXT NOT NULL DEFAULT '',
			commit_sha TEXT NOT NULL DEFAULT '',
			pr_number INTEGER NOT NULL DEFAULT 0,
			workspace_mode_used TEXT NOT NULL DEFAULT '',
			commit_pinned INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT,
			summary_json TEXT NOT NULL DEFAULT '{}',
			error TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scans_repository_started ON scans(repository_id, started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS scanner_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id TEXT NOT NULL,
			scanner_name TEXT NOT NULL,
			status TEXT NOT NULL,
			findings_count INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER,
			detail TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scanner_results_scan ON scanner_results(scan_id)`,
		`CREATE TABLE IF NOT EXISTS findings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			fingerprint TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			severity TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			package_name TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL DEFAULT '',
			line INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			first_seen_scan_id TEXT NOT NULL DEFAULT '',
			last_seen_scan_id TEXT NOT NULL DEFAULT '',
			first_seen_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			UNIQUE(repository_id, fingerprint),
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS finding_instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			finding_id INTEGER NOT NULL,
			scan_id TEXT NOT NULL,
			evidence_redacted TEXT NOT NULL DEFAULT '',
			location_json TEXT NOT NULL DEFAULT '{}',
			raw_metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE,
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_finding_instances_finding_scan ON finding_instances(finding_id, scan_id)`,
		`CREATE TABLE IF NOT EXISTS external_issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			finding_id INTEGER NOT NULL,
			forge_type TEXT NOT NULL DEFAULT 'gitea',
			issue_number INTEGER NOT NULL,
			issue_url TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT 'open',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(finding_id, forge_type, issue_number),
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS lifecycle_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			finding_id INTEGER,
			scan_id TEXT NOT NULL DEFAULT '',
			event_type TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE SET NULL,
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lifecycle_events_finding ON lifecycle_events(finding_id, created_at)`,
	},
	2: {
		`CREATE TABLE IF NOT EXISTS audit_requests (
			audit_id TEXT PRIMARY KEY,
			repo_url TEXT NOT NULL,
			normalized_repo_url TEXT NOT NULL,
			repo_host TEXT NOT NULL DEFAULT '',
			repo_owner TEXT NOT NULL DEFAULT '',
			repo_name TEXT NOT NULL DEFAULT '',
			commit_sha TEXT NOT NULL DEFAULT '',
			default_branch TEXT NOT NULL DEFAULT '',
			audit_depth TEXT NOT NULL DEFAULT 'standard',
			status TEXT NOT NULL,
			risk_score INTEGER NOT NULL DEFAULT 0,
			recommendation TEXT NOT NULL DEFAULT 'unknown',
			started_at TEXT NOT NULL,
			finished_at TEXT,
			summary_json TEXT NOT NULL DEFAULT '{}',
			error TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_requests_started ON audit_requests(started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS audit_findings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			audit_id TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			severity TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL DEFAULT '',
			line INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			evidence_redacted TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			UNIQUE(audit_id, fingerprint),
			FOREIGN KEY (audit_id) REFERENCES audit_requests(audit_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_audit ON audit_findings(audit_id)`,
		`CREATE TABLE IF NOT EXISTS disclosure_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			audit_id TEXT NOT NULL,
			finding_id INTEGER,
			report_type TEXT NOT NULL,
			sensitivity TEXT NOT NULL DEFAULT 'internal_review',
			title TEXT NOT NULL DEFAULT '',
			body_markdown TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 0,
			approved_by_user INTEGER NOT NULL DEFAULT 0,
			submitted_externally INTEGER NOT NULL DEFAULT 0,
			submission_target TEXT NOT NULL DEFAULT '',
			submission_notes TEXT NOT NULL DEFAULT '',
			generated_at TEXT NOT NULL,
			FOREIGN KEY (audit_id) REFERENCES audit_requests(audit_id) ON DELETE CASCADE,
			FOREIGN KEY (finding_id) REFERENCES audit_findings(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_disclosure_reports_audit ON disclosure_reports(audit_id)`,
	},
	3: {
		`ALTER TABLE repo_settings ADD COLUMN enable_health_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_tech_debt_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_reliability_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_maintainability_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_test_gap_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_performance_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_ai_risk_checks INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN health_max_findings INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN health_large_file_lines INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN health_large_function_lines INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN health_max_nesting_depth INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN health_max_function_params INTEGER`,
	},
	4: {
		`CREATE TABLE IF NOT EXISTS scan_graphs (
			scan_id TEXT PRIMARY KEY,
			repository_id INTEGER NOT NULL,
			graph_json TEXT NOT NULL,
			node_count INTEGER NOT NULL DEFAULT 0,
			edge_count INTEGER NOT NULL DEFAULT 0,
			generated_at TEXT NOT NULL,
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_graphs_repository ON scan_graphs(repository_id, generated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS audit_graphs (
			audit_id TEXT PRIMARY KEY,
			graph_json TEXT NOT NULL,
			node_count INTEGER NOT NULL DEFAULT 0,
			edge_count INTEGER NOT NULL DEFAULT 0,
			generated_at TEXT NOT NULL,
			FOREIGN KEY (audit_id) REFERENCES audit_requests(audit_id) ON DELETE CASCADE
		)`,
	},
	5: {
		`ALTER TABLE repo_settings ADD COLUMN enable_code_graph INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN graph_max_nodes INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN graph_max_edges INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN graph_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN graph_include_functions INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN graph_include_findings INTEGER`,
	},
	6: {
		`CREATE TABLE IF NOT EXISTS runner_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL UNIQUE,
			repository_id INTEGER NOT NULL,
			scan_id TEXT,
			job_type TEXT NOT NULL,
			status TEXT NOT NULL,
			runner_mode TEXT NOT NULL DEFAULT 'gitea_actions',
			ref TEXT NOT NULL DEFAULT '',
			commit_sha TEXT NOT NULL DEFAULT '',
			pr_number INTEGER NOT NULL DEFAULT 0,
			policy_snapshot_json TEXT NOT NULL DEFAULT '{}',
			job_spec_json TEXT NOT NULL DEFAULT '{}',
			result_summary_json TEXT NOT NULL DEFAULT '{}',
			error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			expires_at TEXT,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runner_jobs_status ON runner_jobs(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_runner_jobs_scan ON runner_jobs(scan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_runner_jobs_repository ON runner_jobs(repository_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS runner_artifacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			artifact_type TEXT NOT NULL,
			body_json TEXT NOT NULL DEFAULT '{}',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			sha256 TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (job_id) REFERENCES runner_jobs(job_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runner_artifacts_job ON runner_artifacts(job_id)`,
		`CREATE TABLE IF NOT EXISTS runner_nonces (
			nonce TEXT PRIMARY KEY,
			created_at TEXT NOT NULL
		)`,
	},
	7: {
		`ALTER TABLE repo_settings ADD COLUMN enable_govulncheck INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_gosec INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_staticcheck INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN govulncheck_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN gosec_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN staticcheck_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN go_scanner_max_findings INTEGER`,
	},
	8: {
		`ALTER TABLE repo_settings ADD COLUMN enable_hadolint INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN enable_checkov INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN hadolint_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN checkov_timeout_seconds INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN iac_scanner_max_findings INTEGER`,
	},
	9: {
		`ALTER TABLE repo_settings ADD COLUMN scan_profile TEXT`,
	},
	10: {
		`ALTER TABLE repo_settings ADD COLUMN notifications_enabled INTEGER`,
		`ALTER TABLE repo_settings ADD COLUMN notification_min_severity TEXT`,
		`ALTER TABLE repo_settings ADD COLUMN notification_events TEXT`,
		`ALTER TABLE repo_settings ADD COLUMN notification_cooldown_seconds INTEGER`,
	},
	11: {
		`CREATE TABLE IF NOT EXISTS remediation_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id TEXT NOT NULL UNIQUE,
			finding_id INTEGER,
			repository_id INTEGER,
			audit_id TEXT,
			fingerprint TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			severity TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			fix_strategy TEXT NOT NULL DEFAULT '',
			affected_files_json TEXT NOT NULL DEFAULT '[]',
			required_tests_json TEXT NOT NULL DEFAULT '[]',
			validation_commands_json TEXT NOT NULL DEFAULT '[]',
			regression_risk TEXT NOT NULL DEFAULT '',
			fix_complexity TEXT NOT NULL DEFAULT '',
			safe_for_auto_pr INTEGER NOT NULL DEFAULT 0,
			requires_human_review INTEGER NOT NULL DEFAULT 1,
			blocked_reasons_json TEXT NOT NULL DEFAULT '[]',
			advisory INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'proposed',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_plans_finding ON remediation_plans(finding_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_plans_repository ON remediation_plans(repository_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_plans_audit ON remediation_plans(audit_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_plans_fingerprint ON remediation_plans(fingerprint)`,
	},
	12: {
		`CREATE TABLE IF NOT EXISTS patch_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			attempt_id TEXT NOT NULL UNIQUE,
			plan_id TEXT NOT NULL,
			repository_id INTEGER NOT NULL,
			finding_id INTEGER,
			branch_name TEXT NOT NULL DEFAULT '',
			base_ref TEXT NOT NULL DEFAULT '',
			base_commit_sha TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'proposed',
			diff_summary TEXT NOT NULL DEFAULT '',
			files_changed_json TEXT NOT NULL DEFAULT '[]',
			tests_run_json TEXT NOT NULL DEFAULT '[]',
			validation_summary TEXT NOT NULL DEFAULT '',
			pull_request_number INTEGER,
			pull_request_url TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (plan_id) REFERENCES remediation_plans(plan_id) ON DELETE CASCADE,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_patch_attempts_plan ON patch_attempts(plan_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_patch_attempts_repository ON patch_attempts(repository_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_patch_attempts_status ON patch_attempts(status)`,
	},
	13: {
		`ALTER TABLE patch_attempts ADD COLUMN merged_at TEXT`,
		`ALTER TABLE patch_attempts ADD COLUMN merge_commit_sha TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS closure_evidence (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			finding_id INTEGER NOT NULL,
			patch_attempt_id TEXT,
			repository_id INTEGER NOT NULL,
			fingerprint TEXT NOT NULL DEFAULT '',
			merge_commit_sha TEXT NOT NULL DEFAULT '',
			verification_scan_id TEXT NOT NULL DEFAULT '',
			original_source TEXT NOT NULL DEFAULT '',
			scanner_status TEXT NOT NULL DEFAULT '',
			fingerprint_present INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending_rescan',
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_closure_evidence_finding ON closure_evidence(finding_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_closure_evidence_repo_status ON closure_evidence(repository_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_closure_evidence_patch ON closure_evidence(patch_attempt_id)`,
	},
	14: {
		`CREATE TABLE IF NOT EXISTS finding_suppressions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER,
			fingerprint TEXT,
			source TEXT,
			rule_id TEXT,
			category TEXT,
			severity TEXT,
			scope TEXT NOT NULL DEFAULT 'repo',
			reason TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL DEFAULT '',
			expires_at TEXT,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_finding_suppressions_repo_active ON finding_suppressions(repository_id, active)`,
		`CREATE INDEX IF NOT EXISTS idx_finding_suppressions_scope_active ON finding_suppressions(scope, active)`,
		`CREATE INDEX IF NOT EXISTS idx_finding_suppressions_fingerprint ON finding_suppressions(fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_finding_suppressions_rule ON finding_suppressions(rule_id, source)`,
	},
	15: {
		`CREATE TABLE IF NOT EXISTS calibration_rule_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			total_findings INTEGER NOT NULL DEFAULT 0,
			issues_created INTEGER NOT NULL DEFAULT 0,
			suppressions INTEGER NOT NULL DEFAULT 0,
			false_positives INTEGER NOT NULL DEFAULT 0,
			verified_fixes INTEGER NOT NULL DEFAULT 0,
			still_present INTEGER NOT NULL DEFAULT 0,
			last_seen_at TEXT NOT NULL,
			actionable_rate REAL NOT NULL DEFAULT 0,
			false_positive_rate REAL NOT NULL DEFAULT 0,
			recommended_default_action TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			UNIQUE(source, rule_id, category)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calibration_rule_stats_rule ON calibration_rule_stats(rule_id, source)`,
		`CREATE TABLE IF NOT EXISTS calibration_recommendations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope TEXT NOT NULL DEFAULT 'global',
			repository_id INTEGER,
			recommendation_type TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			current_action TEXT NOT NULL DEFAULT '',
			recommended_action TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'proposed',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calibration_recommendations_status ON calibration_recommendations(status, scope)`,
		`CREATE TABLE IF NOT EXISTS issue_reconciliation_runs (
			run_id TEXT PRIMARY KEY,
			repository_id INTEGER NOT NULL,
			preview INTEGER NOT NULL DEFAULT 1,
			item_count INTEGER NOT NULL DEFAULT 0,
			applied INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS issue_reconciliation_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			issue_number INTEGER NOT NULL,
			finding_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT '',
			proposed_action TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (run_id) REFERENCES issue_reconciliation_runs(run_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reconciliation_items_run ON issue_reconciliation_items(run_id)`,
	},
	16: {
		`CREATE INDEX IF NOT EXISTS idx_external_issues_finding_id ON external_issues(finding_id)`,
	},
	17: {
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			display_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_login_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			ip_address TEXT,
			user_agent TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS auth_audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			user_id INTEGER,
			email TEXT,
			ip_address TEXT,
			user_agent TEXT,
			details TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_audit_events_created_at ON auth_audit_events(created_at)`,
	},
}

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	for version := 1; version <= currentSchemaVersion; version++ {
		applied, err := isMigrationApplied(db, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		stmts := migrationStatements[version]
		if len(stmts) == 0 {
			return fmt.Errorf("missing migration statements for version %d", version)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}

		for _, stmt := range stmts {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migration %d failed: %w", version, err)
			}
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))`,
			version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
}

func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return count > 0, nil
}
