package customrules

import (
	"context"
	"strings"
	"testing"
)

const valid = `{"version":1,"rules":[{"id":"CUSTOM_DEBUG","title":"Debug marker in source","category":"quality","severity":"low","path_glob":"src/*.go","pattern":"TODO\\(security\\)"}]}`

func TestScanPathAndLine(t *testing.T) {
	findings, err := Scan(context.Background(), []byte(valid), []File{{Path: "src/app.go", Content: "ok\nTODO(security)\n"}, {Path: "other/app.go", Content: "TODO(security)"}})
	if err != nil || len(findings) != 1 || findings[0].Line != 2 || findings[0].RuleID != "CUSTOM_DEBUG" {
		t.Fatalf("findings=%+v err=%v", findings, err)
	}
}
func TestRejectsExecutablesAndUnknownFields(t *testing.T) {
	for _, bad := range []string{
		`{"version":1,"rules":[{"id":"CUSTOM_RUN","title":"x","category":"security","severity":"high","pattern":".*","command":"rm -rf /"}]}`,
		`{"version":1,"rules":[{"id":"CUSTOM_RUN","title":"x","category":"security","severity":"high","path_glob":"../*","pattern":".*"}]}`,
		`{"version":1,"rules":[{"id":"CUSTOM_RUN","title":"x","category":"security","severity":"high","pattern":"["}]}`,
		`{"version":1,"rules":[]}`,
		strings.Repeat("a", MaxPolicyBytes+1),
	} {
		if _, err := Compile([]byte(bad)); err == nil {
			t.Fatalf("accepted bad policy %.100s", bad)
		}
	}
}
func TestCoverageLimitCannotReturnClean(t *testing.T) {
	_, err := Scan(context.Background(), []byte(valid), []File{{Path: "src/a.go", Content: strings.Repeat("a", MaxFileBytes+1)}})
	if err == nil {
		t.Fatal("expected an explicit incomplete scan error")
	}
}
func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Scan(ctx, []byte(valid), []File{{Path: "src/a.go", Content: "x"}})
	if err == nil {
		t.Fatal("cancel was ignored")
	}
}
func TestKnownSecretNotEchoed(t *testing.T) {
	cfg := `{"version":1,"rules":[{"id":"CUSTOM_SEC","title":"Check","category":"security","severity":"high","pattern":"token="}]}`
	out, err := Scan(context.Background(), []byte(cfg), []File{{Path: "app.go", Content: "token=SUPERSECRET"}})
	if err != nil || len(out) != 1 || strings.Contains(out[0].Description, "SUPERSECRET") {
		t.Fatalf("results=%+v,err=%v", out, err)
	}
}
