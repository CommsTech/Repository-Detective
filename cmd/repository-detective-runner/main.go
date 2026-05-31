package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/analyzers"
	"git.commsnet.org/commstech/bugbot/graph"
	"git.commsnet.org/commstech/bugbot/health"
	"git.commsnet.org/commstech/bugbot/internal/config/envcompat"
	"git.commsnet.org/commstech/bugbot/runner"
	"git.commsnet.org/commstech/bugbot/scanners"
	"github.com/sirupsen/logrus"
)

func envOrCompat(suffix, fallback string) string {
	if value, ok := envcompat.Resolve(suffix); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func main() {
	coreURL := flag.String("core-url", envOrCompat("CORE_URL", ""), "Repository Detective core base URL")
	secret := flag.String("runner-secret", envOrCompat("RUNNER_SHARED_SECRET", ""), "Runner HMAC shared secret")
	workspace := flag.String("workspace", envOrCompat("WORKSPACE", ""), "Checked-out repository directory")
	jobID := flag.String("job-id", envOrCompat("RUNNER_JOB_ID", ""), "Optional job ID (skip claim when set)")
	flag.Parse()

	if *coreURL == "" || *secret == "" || *workspace == "" {
		fmt.Fprintln(os.Stderr, "core-url, runner-secret, and workspace are required")
		os.Exit(2)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	spec, claimedJobID, err := loadJobSpec(ctx, *coreURL, *secret, *jobID)
	if err != nil {
		logger.Fatalf("claim job: %v", err)
	}

	result, err := runner.ExecuteWorkspaceScan(ctx, spec, runner.ExecuteInput{
		WorkspaceDir: *workspace,
		ScannerCfg:   analyzers.ScannersConfigFromSnapshot(scanners.DefaultConfig(), spec.EffectiveSettings),
		HealthCfg:    healthFromPolicy(spec.EffectiveSettings),
		GraphCfg:     graphFromPolicy(spec.EffectiveSettings),
		Logger:       logger,
	})
	if err != nil {
		result.Status = runner.JobStatusFailed
		result.Errors = append(result.Errors, err.Error())
	}

	if err := submitResult(ctx, *coreURL, *secret, claimedJobID, result); err != nil {
		logger.Fatalf("submit result: %v", err)
	}
	logger.Infof("Runner job %s completed with status %s", claimedJobID, result.Status)
}

func loadJobSpec(ctx context.Context, coreURL, secret, jobID string) (runner.JobSpec, string, error) {
	if jobID != "" {
		spec, err := signedGET(ctx, coreURL, secret, "/api/v1/runner/jobs/"+jobID+"/spec")
		if err != nil {
			return runner.JobSpec{}, "", err
		}
		var out runner.JobSpec
		if err := json.Unmarshal(spec, &out); err != nil {
			return runner.JobSpec{}, "", err
		}
		return out, jobID, nil
	}
	body := []byte(`{}`)
	resp, err := signedPOST(ctx, coreURL, secret, "/api/v1/runner/jobs/claim", body)
	if err != nil {
		return runner.JobSpec{}, "", err
	}
	var payload struct {
		Job struct {
			JobID string `json:"job_id"`
		} `json:"job"`
		Spec runner.JobSpec `json:"spec"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return runner.JobSpec{}, "", err
	}
	return payload.Spec, payload.Job.JobID, nil
}

func submitResult(ctx context.Context, coreURL, secret, jobID string, result runner.JobResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = signedPOST(ctx, coreURL, secret, "/api/v1/runner/jobs/"+jobID+"/result", body)
	return err
}

func signedPOST(ctx context.Context, coreURL, secret, path string, body []byte) ([]byte, error) {
	return signedRequest(ctx, http.MethodPost, coreURL, secret, path, body)
}

func signedGET(ctx context.Context, coreURL, secret, path string) ([]byte, error) {
	return signedRequest(ctx, http.MethodGet, coreURL, secret, path, nil)
}

func signedRequest(ctx context.Context, method, coreURL, secret, path string, body []byte) ([]byte, error) {
	ts := fmt.Sprintf("%d", time.Now().UTC().Unix())
	nonce := fmt.Sprintf("rn-%d", time.Now().UTC().UnixNano())
	sig := runner.SignRequest(secret, ts, nonce, method, path, body)
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(coreURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(runner.HeaderTimestamp, ts)
	req.Header.Set(runner.HeaderNonce, nonce)
	req.Header.Set(runner.HeaderSignature, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("core returned %d: %s", resp.StatusCode, string(out))
	}
	return out, nil
}

func healthFromPolicy(p analyzers.PolicySnapshot) health.Config {
	return health.Config{
		Enabled: p.EnableHealthChecks, EnableTechDebt: p.EnableTechDebtChecks,
		EnableReliability: p.EnableReliabilityChecks, EnableMaintainability: p.EnableMaintainabilityChecks,
		EnableTestGap: p.EnableTestGapChecks, EnablePerformance: p.EnablePerformanceChecks,
		EnableAIRisk: p.EnableAIRiskChecks, MaxFindings: p.HealthMaxFindings,
		LargeFileLines: p.HealthLargeFileLines, LargeFunctionLines: p.HealthLargeFunctionLines,
		MaxNestingDepth: p.HealthMaxNestingDepth, MaxFunctionParams: p.HealthMaxFunctionParams,
	}
}

func graphFromPolicy(p analyzers.PolicySnapshot) graph.Config {
	return graph.Config{
		Enabled: p.EnableCodeGraph, MaxNodes: p.GraphMaxNodes, MaxEdges: p.GraphMaxEdges,
		TimeoutSeconds: p.GraphTimeoutSeconds, IncludeFunctions: p.GraphIncludeFunctions,
		IncludeFindings: p.GraphIncludeFindings,
	}
}
