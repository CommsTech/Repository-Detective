package openclaw_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
	_ "modernc.org/sqlite"
)

func TestLiveWifiCollectorHarnessDryRun(t *testing.T) {
	dbPath := "../data/repository-detective.db"
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("live db not available")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id, fingerprint, IFNULL(category,''), IFNULL(source,''), IFNULL(severity,''), confidence, IFNULL(rule_id,''), IFNULL(title,''), IFNULL(file_path,''), line, IFNULL(package_name,'') FROM findings WHERE repository_id=10 AND status='open' LIMIT 500`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var findings []store.Finding
	instances := map[int64]store.FindingInstance{}
	for rows.Next() {
		var f store.Finding
		if err := rows.Scan(&f.ID, &f.Fingerprint, &f.Category, &f.Source, &f.Severity, &f.Confidence, &f.RuleID, &f.Title, &f.FilePath, &f.Line, &f.PackageName); err != nil {
			t.Fatal(err)
		}
		findings = append(findings, f)
		var ev string
		_ = db.QueryRow(`SELECT IFNULL(evidence_redacted,'') FROM finding_instances WHERE finding_id=? ORDER BY id DESC LIMIT 1`, f.ID).Scan(&ev)
		instances[f.ID] = store.FindingInstance{EvidenceRedacted: ev}
	}
	if len(findings) == 0 {
		t.Skip("no open findings")
	}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 4000
	cfg.MaxFindingsPerScan = 12
	cfg.CAH = openclaw.DefaultCAHConfig()
	cfg.CAH.MaxCandidates = 12
	cfg.CAH.TokenBudgetPerScan = 3000
	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID: "dryrun-wifi", Repository: store.Repository{ID: 10, FullName: "commstech/Wifi_Collector"},
		ScanType: openclaw.ScanTypeRepo, Findings: findings, Instances: instances,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]int{}
	emptyEv := 0
	protected := 0
	for _, f := range pkt.Findings {
		sources[f.Source]++
		if f.EvidenceRedacted == "" {
			emptyEv++
		}
		if f.ProtectedFromDowngrade {
			protected++
		}
	}
	t.Logf("OLD wifi review was 10/10 tech_debt empty evidence")
	t.Logf("NEW harness=%s n=%d empty_evidence=%d protected=%d sources=%v", pkt.HarnessVersion, len(pkt.Findings), emptyEv, protected, sources)
	for i, f := range pkt.Findings {
		t.Logf("%d. [%s] %s/%s sev=%s unc=%.2f prot=%v path=%s", i+1, f.CoachLane, f.Source, f.Category, f.Severity, f.UncertaintyScore, f.ProtectedFromDowngrade, f.Path)
	}
	if sources["tech_debt"] == len(pkt.Findings) {
		t.Fatal("packet still entirely tech_debt")
	}
	if protected == 0 {
		t.Fatal("expected some protected coach findings")
	}
	fmt.Fprintf(os.Stderr, "dry-run ok: %d findings sources=%v\n", len(pkt.Findings), sources)
}
