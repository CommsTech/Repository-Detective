package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SQLiteStore) SaveScanGraph(ctx context.Context, record ScanGraphRecord) error {
	if record.ScanID == "" {
		return fmt.Errorf("scan_id is required")
	}
	if record.GeneratedAt.IsZero() {
		record.GeneratedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scan_graphs (scan_id, repository_id, graph_json, node_count, edge_count, generated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(scan_id) DO UPDATE SET
			graph_json = excluded.graph_json,
			node_count = excluded.node_count,
			edge_count = excluded.edge_count,
			generated_at = excluded.generated_at
	`, record.ScanID, record.RepositoryID, string(record.GraphJSON), record.NodeCount, record.EdgeCount, formatTime(record.GeneratedAt))
	if err != nil {
		return fmt.Errorf("save scan graph: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetScanGraph(ctx context.Context, scanID string) (ScanGraphRecord, error) {
	var record ScanGraphRecord
	var graphJSON string
	var generatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT scan_id, repository_id, graph_json, node_count, edge_count, generated_at
		FROM scan_graphs WHERE scan_id = ?
	`, scanID).Scan(&record.ScanID, &record.RepositoryID, &graphJSON, &record.NodeCount, &record.EdgeCount, &generatedAt)
	if err == sql.ErrNoRows {
		return ScanGraphRecord{}, fmt.Errorf("scan graph not found")
	}
	if err != nil {
		return ScanGraphRecord{}, fmt.Errorf("get scan graph: %w", err)
	}
	record.GraphJSON = json.RawMessage(graphJSON)
	record.GeneratedAt = parseTime(generatedAt)
	return record, nil
}

func (s *SQLiteStore) GetLatestScanGraphForRepo(ctx context.Context, repositoryID int64) (ScanGraphRecord, error) {
	var record ScanGraphRecord
	var graphJSON string
	var generatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT scan_id, repository_id, graph_json, node_count, edge_count, generated_at
		FROM scan_graphs WHERE repository_id = ?
		ORDER BY generated_at DESC LIMIT 1
	`, repositoryID).Scan(&record.ScanID, &record.RepositoryID, &graphJSON, &record.NodeCount, &record.EdgeCount, &generatedAt)
	if err == sql.ErrNoRows {
		return ScanGraphRecord{}, fmt.Errorf("scan graph not found")
	}
	if err != nil {
		return ScanGraphRecord{}, fmt.Errorf("get latest scan graph: %w", err)
	}
	record.GraphJSON = json.RawMessage(graphJSON)
	record.GeneratedAt = parseTime(generatedAt)
	return record, nil
}

func (s *SQLiteStore) SaveAuditGraph(ctx context.Context, record AuditGraphRecord) error {
	if record.AuditID == "" {
		return fmt.Errorf("audit_id is required")
	}
	if record.GeneratedAt.IsZero() {
		record.GeneratedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_graphs (audit_id, graph_json, node_count, edge_count, generated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(audit_id) DO UPDATE SET
			graph_json = excluded.graph_json,
			node_count = excluded.node_count,
			edge_count = excluded.edge_count,
			generated_at = excluded.generated_at
	`, record.AuditID, string(record.GraphJSON), record.NodeCount, record.EdgeCount, formatTime(record.GeneratedAt))
	if err != nil {
		return fmt.Errorf("save audit graph: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetAuditGraph(ctx context.Context, auditID string) (AuditGraphRecord, error) {
	var record AuditGraphRecord
	var graphJSON string
	var generatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT audit_id, graph_json, node_count, edge_count, generated_at
		FROM audit_graphs WHERE audit_id = ?
	`, auditID).Scan(&record.AuditID, &graphJSON, &record.NodeCount, &record.EdgeCount, &generatedAt)
	if err == sql.ErrNoRows {
		return AuditGraphRecord{}, fmt.Errorf("audit graph not found")
	}
	if err != nil {
		return AuditGraphRecord{}, fmt.Errorf("get audit graph: %w", err)
	}
	record.GraphJSON = json.RawMessage(graphJSON)
	record.GeneratedAt = parseTime(generatedAt)
	return record, nil
}
