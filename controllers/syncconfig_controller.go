package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
)

type (
	// SyncConfigReconciler reconciles SyncConfig objects
	SyncConfigReconciler struct {
		Client            client.Client
		Log               logr.Logger
		Scheme            *runtime.Scheme
		ReconcileInterval time.Duration
		WatchNamespace    string
		// NamespaceScope limits creations and deletions of sync items to this namespace, provided the selector still matches.
		// If empty, the sync applies to all selector-matching namespaces.
		NamespaceScope string
	}
	// ReconciliationContext holds the parameters of a single SyncConfig reconciliation
	ReconciliationContext struct {
		ctx              context.Context
		cfg              *syncv1alpha1.SyncConfig
		matchNamesRegex  []*regexp.Regexp
		ignoreNamesRegex []*regexp.Regexp
		nsSelector       labels.Selector
		syncCount        int64
		deleteCount      int64
		failCount        int64
	}
)

// SetupWithManager configures this reconciler with the given manager
func (r *SyncConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.SyncConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs/status,verbs=get;update;patch

// Reconcile retrieves a SyncConfig from the given reconcile request
func (r *SyncConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	syncConfig := &syncv1alpha1.SyncConfig{}

	err := r.Client.Get(ctx, req.NamespacedName, syncConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("SyncConfig not found, ignoring reconcile.", "SyncConfig", req.NamespacedName)
			return ctrl.Result{Requeue: false}, nil
		}
		r.Log.Error(err, "Could not retrieve SyncConfig.", "SyncConfig", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: r.ReconcileInterval}, err
	}

	result, err := r.DoReconcile(ctx, syncConfig)
	result.RequeueAfter = r.ReconcileInterval
	return result, err
}

// DoReconcile is the actual reconciliation of the given SyncConfig
func (r *SyncConfigReconciler) DoReconcile(ctx context.Context, syncConfig *syncv1alpha1.SyncConfig) (ctrl.Result, error) {
	rc := &ReconciliationContext{
		ctx: ctx,
		cfg: syncConfig,
	}
	r.Log.Info("Reconciling", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	err := rc.validateSpec()
	if err != nil {
		rc.SetStatusCondition(CreateStatusConditionInvalid(err))
		rc.SetStatusCondition(CreateStatusConditionReady(false))
		return ctrl.Result{}, r.updateStatus(rc)
	}
	rc.SetStatusIfExisting(syncv1alpha1.ConditionInvalid, metav1.ConditionFalse)

	namespaces, fetchErr := r.fetchNamespaces(rc)
	if fetchErr != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, r.updateStatus(rc)
	}
	filteredNamespaces := rc.filterNamespaces(namespaces)

	for _, targetNamespace := range filteredNamespaces {
		if targetNamespace.Status.Phase == corev1.NamespaceActive {
			r.deleteItems(rc, targetNamespace)
			r.syncItems(rc, targetNamespace)
		}
	}
	if rc.failCount > 0 {
		r.Log.V(1).Info("Encountered errors", "err_count", rc.failCount)
	}
	if rc.isReconcileFailed() {
		rc.SetStatusCondition(CreateStatusConditionReady(false))
		rc.SetStatusCondition(CreateStatusConditionErrored(fmt.Errorf("could not sync or delete any items")))
	} else {
		rc.SetStatusCondition(CreateStatusConditionReady(true))
	}
	return ctrl.Result{}, r.updateStatus(rc)
}

func (r *SyncConfigReconciler) syncItems(rc *ReconciliationContext, targetNamespace corev1.Namespace) {
	for _, item := range rc.cfg.Spec.SyncItems {
		obj := item.DeepCopy()
		obj.SetNamespace(targetNamespace.Name)
		replaceProjectName(targetNamespace.Name, obj.Object)

		err := r.syncItem(rc, obj, rc.cfg.Spec.ForceRecreate)
		if err != nil {
			r.Log.Error(err, "Error syncing object", getLoggingKeysAndValues(obj)...)
			rc.IncrementFailCount()
		} else {
			rc.IncrementSyncCount()
		}
	}
	return
}
func (r *SyncConfigReconciler) syncItem(rc *ReconciliationContext, obj *unstructured.Unstructured, force bool) error {
	l := r.Log.
		WithValues(getLoggingKeysAndValues(obj)...).
		WithValues(getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	l.V(2).Info("Syncing object")

	found := &unstructured.Unstructured{}
	found.SetKind(obj.GetKind())
	found.SetAPIVersion(obj.GetAPIVersion())
	found.SetName(obj.GetName())
	found.SetNamespace(obj.GetNamespace())

	op, err := controllerutil.CreateOrUpdate(rc.ctx, r.Client, found, func() error {
		copyInto(found, obj)
		return nil
	})
	if op != controllerutil.OperationResultNone {
		l.Info("Modified object")
	}

	if apierrors.IsInvalid(err) && force {
		err = r.recreateObject(rc, obj)
		if err != nil {
			return err
		}
		l.Info("Force recreated object")
	}

	return err
}

func (r *SyncConfigReconciler) deleteItems(rc *ReconciliationContext, targetNamespace corev1.Namespace) {
	for _, deleteItem := range rc.cfg.Spec.DeleteItems {
		r.Log.V(1).Info("Deleting", "item", deleteItem)
		deleteObj := deleteItem.ToDeleteObj(targetNamespace.Name)

		propagationPolicy := metav1.DeletePropagationBackground
		err := r.Client.Delete(rc.ctx, deleteObj, &client.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				rc.IncrementDeleteCount()
				r.Log.WithValues(getLoggingKeysAndValues(deleteObj)...).Info("Error deleting object", "error", err)
			}
		} else {
			r.Log.Info("Deleted", getLoggingKeysAndValues(deleteObj)...)
			rc.IncrementDeleteCount()
		}
	}
	return
}

func (r *SyncConfigReconciler) fetchNamespaces(rc *ReconciliationContext) ([]corev1.Namespace, error) {
	if r.NamespaceScope != "" {
		return r.fetchSingleNamespace(rc)
	}
	namespaceList := &corev1.NamespaceList{}
	err := r.Client.List(rc.ctx, namespaceList)
	if err != nil {
		return []corev1.Namespace{}, err
	}
	return namespaceList.Items, nil
}

func (r *SyncConfigReconciler) fetchSingleNamespace(rc *ReconciliationContext) ([]corev1.Namespace, error) {
	namespace := corev1.Namespace{}
	err := r.Client.Get(rc.ctx, types.NamespacedName{Name: r.NamespaceScope}, &namespace)
	if err != nil {
		return []corev1.Namespace{}, err
	}
	return []corev1.Namespace{namespace}, nil
}

func (r *SyncConfigReconciler) recreateObject(rc *ReconciliationContext, obj *unstructured.Unstructured) error {
	obj.SetResourceVersion("")

	propagationPolicy := metav1.DeletePropagationForeground
	err := r.Client.Delete(rc.ctx, obj, &client.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return err
	}

	err = r.Client.Create(context.Background(), obj)
	if err != nil {
		return err
	}

	return nil
}
