package main

import (
	"context"
	"fmt"

	"git.commsnet.org/commstech/bugbot/reconcile"
	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
)

var reconcileEngine *reconcile.Engine

type reconcileForgeBridge struct{}

func (reconcileForgeBridge) AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error {
	if issueManager == nil {
		return fmt.Errorf("issue manager unavailable")
	}
	forge := issueManager.ForgeFor("gitea")
	if forge == nil {
		return fmt.Errorf("forge unavailable")
	}
	return forge.AddIssueLabels(ctx, owner, repo, issueNumber, labels)
}

func (reconcileForgeBridge) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	if issueManager == nil {
		return fmt.Errorf("issue manager unavailable")
	}
	forge := issueManager.ForgeFor("gitea")
	if forge == nil {
		return fmt.Errorf("forge unavailable")
	}
	return forge.CreateIssueComment(ctx, owner, repo, issueNumber, body)
}

func (reconcileForgeBridge) CloseIssue(ctx context.Context, owner, repo string, issueNumber int) error {
	if giteaClient == nil {
		return fmt.Errorf("gitea client unavailable")
	}
	return giteaClient.CloseIssue(ctx, owner, repo, issueNumber)
}

func (reconcileForgeBridge) AnnotateCalibration(ctx context.Context, forgeType, owner, repo string, issueNumber int, falsePositive bool, reason string) error {
	if issueManager == nil {
		return fmt.Errorf("issue manager unavailable")
	}
	return issueManager.AnnotateCalibration(ctx, forgeType, owner, repo, issueNumber, falsePositive, reason)
}

func initReconcileEngine() {
	if bugbotStore == nil || !config.IssueReconciliationEnabled {
		reconcileEngine = nil
		return
	}
	basePath := config.PublicURL
	if config.UIEnabled {
		basePath = config.PublicURL + "/ui"
	}
	reconcileEngine = reconcile.NewEngine(bugbotStore, suppressionMatcher, reconcileForgeBridge{}, reconcile.Config{
		Enabled:             true,
		Comment:             config.IssueReconciliationComment,
		CloseVerified:       config.IssueReconciliationCloseVerified,
		MaxCommentsPerIssue: config.IssueReconciliationMaxCommentsPerIssue,
		PublicBasePath:      basePath,
	})
}

type reconcileBridge struct{}

func (reconcileBridge) Preview(c *gin.Context, repositoryID int64) (reconcile.Result, error) {
	if reconcileEngine == nil {
		return reconcile.Result{}, fmt.Errorf("issue reconciliation disabled")
	}
	return reconcileEngine.Preview(c.Request.Context(), repositoryID)
}

func (reconcileBridge) Apply(c *gin.Context, repositoryID int64) (reconcile.Result, error) {
	if reconcileEngine == nil {
		return reconcile.Result{}, fmt.Errorf("issue reconciliation disabled")
	}
	return reconcileEngine.Apply(c.Request.Context(), repositoryID)
}

func (reconcileBridge) GetRun(c *gin.Context, runID string) (store.ReconciliationRun, []store.ReconciliationItemRecord, error) {
	if bugbotStore == nil {
		return store.ReconciliationRun{}, nil, fmt.Errorf("database disabled")
	}
	return bugbotStore.GetReconciliationRun(c.Request.Context(), runID)
}
