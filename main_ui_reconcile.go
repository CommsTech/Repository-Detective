package main

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/reconcile"
)

type uiReconcileBridge struct{}

func (uiReconcileBridge) Preview(ctx context.Context, repositoryID int64) (any, error) {
	if reconcileEngine == nil {
		return nil, nil
	}
	return reconcileEngine.Preview(ctx, repositoryID)
}

func (uiReconcileBridge) Apply(ctx context.Context, repositoryID int64) (any, error) {
	if reconcileEngine == nil {
		return reconcile.Result{}, nil
	}
	return reconcileEngine.Apply(ctx, repositoryID)
}
