package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.commsnet.org/commstech/repository-detective/runner"
	"github.com/sirupsen/logrus"
)

func main() {
	mode := flag.String("mode", envOr("RUNNER_MODE", "worker"), "worker (daemon) or once (single job with existing workspace)")
	coreURL := flag.String("core-url", envOr("CORE_URL", ""), "Repository Detective core base URL")
	secret := flag.String("runner-secret", envOr("RUNNER_SHARED_SECRET", ""), "Runner HMAC shared secret")
	workspace := flag.String("workspace", envOr("WORKSPACE", ""), "Workspace directory (once mode)")
	jobID := flag.String("job-id", envOr("RUNNER_JOB_ID", ""), "Optional job ID (once mode)")
	flag.Parse()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	switch *mode {
	case "worker":
		runWorker(logger)
	case "once":
		if err := runOnce(logger, *coreURL, *secret, *workspace, *jobID); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (use worker or once)\n", *mode)
		os.Exit(2)
	}
}

func runWorker(logger *logrus.Logger) {
	cfg := runner.LoadWorkerConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	worker, err := runner.NewWorker(cfg, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := worker.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func runOnce(logger *logrus.Logger, coreURL, secret, workspace, jobID string) error {
	if coreURL == "" || secret == "" || workspace == "" {
		return fmt.Errorf("core-url, runner-secret, and workspace are required in once mode")
	}
	client := runner.NewClient(coreURL, secret)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var spec runner.JobSpec
	var claimedJobID string
	var err error
	if jobID != "" {
		spec, err = client.GetJobSpec(ctx, jobID)
		claimedJobID = jobID
	} else {
		claimedJobID, spec, err = client.ClaimNextJob(ctx)
	}
	if err != nil {
		return fmt.Errorf("claim job: %w", err)
	}

	scannerCfg, healthCfg, graphCfg := runner.PolicyConfigsFromSpec(spec)
	result, execErr := runner.ExecuteJob(ctx, spec, runner.JobExecuteInput{
		WorkspaceDir: workspace,
		ScannerCfg:   scannerCfg,
		HealthCfg:    healthCfg,
		GraphCfg:     graphCfg,
		Logger:       logger,
	})
	if execErr != nil && result.Status != runner.JobStatusFailed {
		result.Status = runner.JobStatusFailed
		result.Errors = append(result.Errors, runner.RedactLogLine(execErr.Error()))
	}
	if err := client.SubmitResult(ctx, claimedJobID, result); err != nil {
		return fmt.Errorf("submit result: %w", err)
	}
	logger.Infof("runner once job %s completed status=%s", claimedJobID, result.Status)
	return nil
}

func envOr(suffix, fallback string) string {
	cfg := runner.LoadWorkerConfigFromEnv()
	switch suffix {
	case "CORE_URL":
		if cfg.CoreURL != "" {
			return cfg.CoreURL
		}
	case "RUNNER_SHARED_SECRET":
		if cfg.SharedSecret != "" {
			return cfg.SharedSecret
		}
	}
	return fallback
}
