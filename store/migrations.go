package store

import (
	"database/sql"
	"fmt"
)

const currentSchemaVersion = 25

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
	18: {
		`CREATE INDEX IF NOT EXISTS idx_finding_instances_scan_id ON finding_instances(scan_id)`,
	},
	19: {
		`CREATE TABLE IF NOT EXISTS project_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			description TEXT NOT NULL DEFAULT '',
			primary_repository_id INTEGER,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (primary_repository_id) REFERENCES repositories(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_group_repositories (
			project_group_id INTEGER NOT NULL,
			repository_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (project_group_id, repository_id),
			FOREIGN KEY (project_group_id) REFERENCES project_groups(id) ON DELETE CASCADE,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS sbom_artifacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER,
			scan_id TEXT NOT NULL DEFAULT '',
			format TEXT NOT NULL DEFAULT '',
			package_count INTEGER NOT NULL DEFAULT 0,
			vuln_count INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			artifact_path TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sbom_artifacts_scan ON sbom_artifacts(scan_id)`,
	},
	20: {
		`CREATE TABLE IF NOT EXISTS learning_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			scan_id TEXT NOT NULL DEFAULT '',
			finding_id INTEGER,
			fingerprint TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			event_type TEXT NOT NULL,
			evidence_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			created_by TEXT NOT NULL DEFAULT '',
			confidence_delta REAL NOT NULL DEFAULT 0,
			idempotency_key TEXT NOT NULL DEFAULT '',
			UNIQUE(idempotency_key),
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_events_repo ON learning_events(repository_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_events_rule ON learning_events(repository_id, source, rule_id)`,
		`CREATE TABLE IF NOT EXISTS repo_calibration_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER,
			project_group_id INTEGER,
			scope TEXT NOT NULL DEFAULT 'repo',
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			path_pattern TEXT NOT NULL DEFAULT '',
			finding_category TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL DEFAULT 'no_change',
			reason TEXT NOT NULL DEFAULT '',
			evidence_count INTEGER NOT NULL DEFAULT 0,
			false_positive_rate REAL NOT NULL DEFAULT 0,
			true_positive_rate REAL NOT NULL DEFAULT 0,
			duplicate_rate REAL NOT NULL DEFAULT 0,
			expires_at TEXT,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			recommendation_id INTEGER,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_repo_calibration_rules_repo ON repo_calibration_rules(repository_id, active)`,
		`CREATE TABLE IF NOT EXISTS rule_reliability_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER,
			project_group_id INTEGER,
			source TEXT NOT NULL DEFAULT '',
			rule_id TEXT NOT NULL DEFAULT '',
			scans_seen INTEGER NOT NULL DEFAULT 0,
			findings_seen INTEGER NOT NULL DEFAULT 0,
			true_positive_count INTEGER NOT NULL DEFAULT 0,
			false_positive_count INTEGER NOT NULL DEFAULT 0,
			resolved_verified_count INTEGER NOT NULL DEFAULT 0,
			duplicate_count INTEGER NOT NULL DEFAULT 0,
			reappeared_count INTEGER NOT NULL DEFAULT 0,
			issue_created_count INTEGER NOT NULL DEFAULT 0,
			issue_closed_count INTEGER NOT NULL DEFAULT 0,
			scanner_failure_count INTEGER NOT NULL DEFAULT 0,
			last_seen_at TEXT NOT NULL,
			reliability_score REAL NOT NULL DEFAULT 0,
			actionability_score REAL NOT NULL DEFAULT 0,
			UNIQUE(repository_id, source, rule_id)
		)`,
		`CREATE TABLE IF NOT EXISTS scanner_health_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			scan_id TEXT NOT NULL,
			scanner TEXT NOT NULL,
			status TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			finding_count INTEGER NOT NULL DEFAULT 0,
			error_class TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scanner_health_repo ON scanner_health_history(repository_id, scanner, created_at)`,
		`ALTER TABLE findings ADD COLUMN structural_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE findings ADD COLUMN canonical_finding_id INTEGER`,
		`ALTER TABLE findings ADD COLUMN calibration_note TEXT NOT NULL DEFAULT ''`,
	},
	21: {
		`CREATE TABLE IF NOT EXISTS container_image_references (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			image TEXT NOT NULL,
			tag TEXT NOT NULL DEFAULT '',
			digest TEXT NOT NULL DEFAULT '',
			target_type TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL DEFAULT '',
			line INTEGER NOT NULL DEFAULT 0,
			service_name TEXT NOT NULL DEFAULT '',
			mutable_tag INTEGER NOT NULL DEFAULT 0,
			private_registry INTEGER NOT NULL DEFAULT 0,
			last_scan_id TEXT NOT NULL DEFAULT '',
			last_digest TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			meta_json TEXT NOT NULL DEFAULT '{}',
			UNIQUE(repository_id, image, file_path, line),
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_container_image_refs_repo ON container_image_references(repository_id)`,
		`CREATE TABLE IF NOT EXISTS container_image_scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			scan_id TEXT NOT NULL DEFAULT '',
			runner_job_id TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL,
			image_digest TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'queued',
			vuln_count INTEGER NOT NULL DEFAULT 0,
			sbom_path TEXT NOT NULL DEFAULT '',
			sbom_format TEXT NOT NULL DEFAULT '',
			coverage_json TEXT NOT NULL DEFAULT '{}',
			warnings_json TEXT NOT NULL DEFAULT '[]',
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_container_image_scans_repo ON container_image_scans(repository_id, created_at)`,
	},
	22: {
		`CREATE TABLE IF NOT EXISTS ai_advisory_reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			review_id TEXT NOT NULL UNIQUE,
			scan_id TEXT NOT NULL DEFAULT '',
			repository_id INTEGER NOT NULL,
			scan_type TEXT NOT NULL DEFAULT 'repo',
			status TEXT NOT NULL DEFAULT 'queued',
			findings_sent INTEGER NOT NULL DEFAULT 0,
			redaction_count INTEGER NOT NULL DEFAULT 0,
			recommendations_count INTEGER NOT NULL DEFAULT 0,
			overall_assessment TEXT NOT NULL DEFAULT '',
			packet_json TEXT NOT NULL DEFAULT '{}',
			response_json TEXT NOT NULL DEFAULT '{}',
			error_message TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_advisory_reviews_scan ON ai_advisory_reviews(scan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_advisory_reviews_repo ON ai_advisory_reviews(repository_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS ai_advisory_recommendations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			review_id TEXT NOT NULL,
			finding_fingerprint TEXT NOT NULL,
			classification TEXT NOT NULL DEFAULT '',
			suggested_action TEXT NOT NULL DEFAULT '',
			suggested_severity TEXT NOT NULL DEFAULT '',
			suggested_confidence TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			evidence_gaps_json TEXT NOT NULL DEFAULT '[]',
			operator_status TEXT NOT NULL DEFAULT 'pending',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (review_id) REFERENCES ai_advisory_reviews(review_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_advisory_recs_review ON ai_advisory_recommendations(review_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_advisory_recs_fp ON ai_advisory_recommendations(finding_fingerprint)`,
	},
	23: {
		`CREATE TABLE IF NOT EXISTS platform_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			settings_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL,
			updated_by TEXT NOT NULL DEFAULT ''
		)`,
	},
	// UI responsiveness: indexes for dashboard + repo-control hot paths on large operator DBs.
	24: {
		`CREATE INDEX IF NOT EXISTS idx_finding_instances_created_at ON finding_instances(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_status ON findings(status)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_status_severity ON findings(status, severity)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_status_repo ON findings(status, repository_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scans_status_started ON scans(status, started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_scans_trigger_started ON scans(trigger_type, started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_scanner_results_status ON scanner_results(status)`,
		`CREATE INDEX IF NOT EXISTS idx_scanner_results_scan_status ON scanner_results(scan_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_external_issues_state_finding ON external_issues(state, finding_id)`,
	},
	// Operator acceptance evidence (webhook delivery, first-scan proofs) — RD-017A.
	25: {
		`CREATE TABLE IF NOT EXISTS operator_evidence (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL
		)`,
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
				// Rollback after failed statement; ignore Rollback error (tx may already be invalid).
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
