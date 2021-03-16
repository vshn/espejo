package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
		Watches(&source.Kind{Type: &corev1.Namespace{}}, handler.EnqueueRequestsFromMapFunc(r.Map)).
		Complete(r)
}

// Map transforms the watched objects into a list of SyncConfig to enqueue for later reconciliation.
func (r *SyncConfigReconciler) Map(object client.Object) (reqs []reconcile.Request) {
	configList := &syncv1alpha1.SyncConfigList{}
	ctx := context.Background()

	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: object.GetName()}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("Namespace does not exist, ignoring reconcile.", "namespace", object.GetName())
		} else {
			r.Log.Info("Could not get namespace status.", "namespace", object.GetName(), "error", err.Error())
		}
		return
	}
	if ns.Status.Phase != corev1.NamespaceActive {
		r.Log.V(1).Info("Namespace is not active, ignoring reconcile.", "namespace", ns.Name, "phase", ns.Status.Phase)
		return
	}

	r.Log.Info("Reconciling from Namespace event", "namespace", object.GetName())
	err = r.Client.List(ctx, configList, &client.ListOptions{Namespace: r.WatchNamespace})
	if err != nil {
		r.Log.Error(err, "Could not get list of SyncConfig")
		return
	}
	for _, cfg := range configList.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
		}})
	}
	return
}

// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// Reconcile processes the given SyncConfig.
func (r *SyncConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, returnErr error) {
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

	rc := &ReconciliationContext{
		ctx: ctx,
		cfg: syncConfig,
	}
	r.Log.Info("Reconciling", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	err = rc.validateSpec()
	if err != nil {
		rc.SetStatusCondition(CreateStatusConditionInvalid(err))
		rc.SetStatusCondition(CreateStatusConditionReady(false))
		return ctrl.Result{Requeue: false}, r.updateStatus(rc)
	}
	rc.SetStatusIfExisting(syncv1alpha1.ConditionInvalid, metav1.ConditionFalse)

	namespaces, reconcileErr := r.getNamespaces(rc)
	if reconcileErr != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, r.updateStatus(rc)
	}

	for _, targetNamespace := range namespaces {
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
	return ctrl.Result{RequeueAfter: r.ReconcileInterval}, r.updateStatus(rc)
}

func (r *SyncConfigReconciler) syncItems(rc *ReconciliationContext, targetNamespace corev1.Namespace) {
	for _, item := range rc.cfg.Spec.SyncItems {
		cp := unstructured.Unstructured(*item.DeepCopy())
		unstructObj := &cp
		unstructObj.SetNamespace(targetNamespace.Name)

		replaceProjectName(targetNamespace.Name, unstructObj.Object)

		err := r.Client.Create(rc.ctx, unstructObj)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				rc.IncrementFailCount()
				r.Log.Error(err, "Error creating", getLoggingKeysAndValues(unstructObj)...)
				continue
			}
			existingObj := &unstructured.Unstructured{}
			existingObj.SetKind(unstructObj.GetKind())
			existingObj.SetAPIVersion(unstructObj.GetAPIVersion())
			err = r.Client.Get(rc.ctx,
				types.NamespacedName{
					Name:      unstructObj.GetName(),
					Namespace: unstructObj.GetNamespace()},
				existingObj)
			if err != nil {
				rc.IncrementFailCount()
				continue
			}
			unstructObj.SetResourceVersion(existingObj.GetResourceVersion())
			err = r.Client.Update(rc.ctx, unstructObj)
			if err != nil {
				if apierrors.IsInvalid(err) && rc.cfg.Spec.ForceRecreate {
					err = r.recreateObject(rc, unstructObj)
					if err != nil {
						rc.IncrementFailCount()
						r.Log.Error(err, "Error recreating", getLoggingKeysAndValues(unstructObj)...)
					}
					rc.IncrementSyncCount()
					r.Log.Info("Recreated", getLoggingKeysAndValues(unstructObj)...)
				} else {
					rc.IncrementFailCount()
					r.Log.Error(err, "Error updating", getLoggingKeysAndValues(unstructObj)...)
				}
			} else {
				rc.IncrementSyncCount()
				r.Log.Info("Updated", getLoggingKeysAndValues(unstructObj)...)
			}
		} else {
			rc.IncrementSyncCount()
			r.Log.Info("Created", getLoggingKeysAndValues(unstructObj)...)
		}

	}
	return
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

func (r *SyncConfigReconciler) getNamespaces(rc *ReconciliationContext) (namespaces []corev1.Namespace, returnErr error) {
	namespaceList := &corev1.NamespaceList{}
	err := r.Client.List(rc.ctx, namespaceList)
	if err != nil {
		return []corev1.Namespace{}, err
	}

	return rc.filterNamespaces(namespaceList.Items), nil
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
