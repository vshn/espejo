package syncconfig

import (
	"context"

	"github.com/vshn/espejo/pkg/apis/sync/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileSyncConfig) handle(ctx context.Context, syncConfig *v1alpha1.SyncConfig, request reconcile.Request) error {

	return nil
}
