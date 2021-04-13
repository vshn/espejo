package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
)

type (
	// NamespaceReconciler reconciles SyncConfigs from namespace events
	NamespaceReconciler struct {
		Client                  client.Client
		Log                     logr.Logger
		Scheme                  *runtime.Scheme
		WatchNamespace          string
		NewSyncConfigReconciler func() *SyncConfigReconciler
	}
	// NamespaceReconciliationContext holds parameters relevant for a single reconcile
	NamespaceReconciliationContext struct {
		namespace *corev1.Namespace
		ctx       context.Context
	}
)

// SetupWithManager configures this reconciler with the given manager
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get

// Reconcile processes the given namespace event request.
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ns := &corev1.Namespace{}
	rc := &NamespaceReconciliationContext{
		namespace: ns,
		ctx:       ctx,
	}
	name := req.Name
	err := r.Client.Get(ctx, types.NamespacedName{Name: name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("Namespace does not exist, ignoring reconcile.", "namespace", name)
			return ctrl.Result{}, nil
		}
		r.Log.Info("Could not fetch namespace", "namespace", name, "error", err.Error())
		return ctrl.Result{}, err
	}
	if ns.Status.Phase != corev1.NamespaceActive {
		r.Log.V(1).Info("Namespace is not active, ignoring reconcile.", "namespace", ns.Name, "phase", ns.Status.Phase)
		return ctrl.Result{}, nil
	}

	configList := &syncv1alpha1.SyncConfigList{}

	r.Log.Info("Reconciling from Namespace event", "namespace", name)
	var options []client.ListOption
	if r.WatchNamespace != "" {
		options = append(options, client.InNamespace(r.WatchNamespace))
	}
	err = r.Client.List(ctx, configList, options...)
	if err != nil {
		r.Log.Error(err, "Could not get list of SyncConfig")
		return ctrl.Result{}, err
	}

	return r.reconcileSyncConfigsForNamespace(rc, configList)
}

func (r *NamespaceReconciler) reconcileSyncConfigsForNamespace(rc *NamespaceReconciliationContext, configList *syncv1alpha1.SyncConfigList) (ctrl.Result, error) {
	scr := r.NewSyncConfigReconciler()
	scr.NamespaceScope = rc.namespace.Name
	for _, cfg := range configList.Items {
		if result, err := scr.DoReconcile(rc.ctx, &cfg); err != nil {
			return result, err
		}
	}
	return ctrl.Result{}, nil
}
