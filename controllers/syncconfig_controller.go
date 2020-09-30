/*
Licensed under the Apache License, Version 2.0 (the "License");
http://www.apache.org/licenses/LICENSE-2.0
*/

package controllers

import (
	"context"
	"fmt"
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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
)

type (
	// SyncConfigReconciler reconciles a SyncConfig object
	SyncConfigReconciler struct {
		Client            client.Client
		Log               logr.Logger
		Scheme            *runtime.Scheme
		ReconcileInterval time.Duration
		WatchNamespace    string
	}
	// ReconciliationContext holds the parameters of a reconciliation
	ReconciliationContext struct {
		ctx context.Context
		cfg *syncv1alpha1.SyncConfig
	}
)

func (r *SyncConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.SyncConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: r}).
		Complete(r)
}

// Map transforms the watched objects into a list of SyncConfig to enqueue for later reconciliation.
func (r *SyncConfigReconciler) Map(object handler.MapObject) (reqs []reconcile.Request) {
	configList := &syncv1alpha1.SyncConfigList{}
	ctx := context.Background()

	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: object.Meta.GetName()}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("Namespace does not exist, ignoring reconcile.", "namespace", object.Meta.GetName())
		} else {
			r.Log.Info("Could not get namespace status.", "namespace", object.Meta.GetName(), "error", err.Error())
		}
		return
	}
	if ns.Status.Phase != corev1.NamespaceActive {
		r.Log.V(1).Info("Namespace is not active, ignoring reconcile.", "namespace", ns.Name, "phase", ns.Status.Phase)
		return
	}

	r.Log.Info("Reconciling from Namespace event", "namespace", object.Meta.GetName())
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

func (r *SyncConfigReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, returnErr error) {
	ctx := context.Background()
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
	errCount := 0
	r.Log.Info("Reconciling", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	namespaces, reconcileErr := r.getNamespaces(rc)
	if reconcileErr != nil {
		returnErr = r.updateStatus(rc, reconcileErr)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, returnErr
	}
	for _, targetNamespace := range namespaces {
		if targetNamespace.Status.Phase == corev1.NamespaceActive {
			errCount += r.deleteItems(rc, targetNamespace)
			errCount += r.syncItems(rc, targetNamespace)
		}
	}
	if errCount > 0 {
		r.Log.V(1).Info("Encountered errors", "err_count", errCount)
		returnErr = fmt.Errorf("%v errors occured during reconcilement", errCount)
	}
	returnErr = r.updateStatus(rc, returnErr)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, returnErr
}

func (r *SyncConfigReconciler) updateStatus(rc *ReconciliationContext, reconcileErr error) error {
	if reconcileErr != nil {
		rc.cfg.Status.ReconcileError = reconcileErr.Error()
	} else {
		// clear errors
		rc.cfg.Status.ReconcileError = ""
	}
	err := r.Client.Status().Update(rc.ctx, rc.cfg)
	if err != nil {
		r.Log.Error(err, "Could not update SyncConfig.", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
		return err
	}
	r.Log.Info("Updated SyncConfig status.", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	return nil
}

func (r *SyncConfigReconciler) syncItems(rc *ReconciliationContext, targetNamespace corev1.Namespace) (errCount int) {
	for _, item := range rc.cfg.Spec.SyncItems {
		unstructObj := item.DeepCopy()
		unstructObj.SetNamespace(targetNamespace.Name)

		replaceProjectName(targetNamespace.Name, unstructObj.Object)

		err := r.Client.Create(rc.ctx, unstructObj)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				errCount++
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
				errCount++
				continue
			}
			unstructObj.SetResourceVersion(existingObj.GetResourceVersion())
			err = r.Client.Update(rc.ctx, unstructObj)
			if err != nil {
				if apierrors.IsInvalid(err) && rc.cfg.Spec.ForceRecreate {
					r.Log.Info("Recreating", getLoggingKeysAndValues(unstructObj)...)
					err = r.recreateObject(rc, unstructObj)
					if err != nil {
						errCount++
						r.Log.Error(err, "Error recreating", getLoggingKeysAndValues(unstructObj)...)
					}
				} else {
					errCount++
					r.Log.Error(err, "Error updating", getLoggingKeysAndValues(unstructObj)...)
				}
			} else {
				r.Log.Info("Updated", getLoggingKeysAndValues(unstructObj)...)
			}
		} else {
			r.Log.Info("Created", getLoggingKeysAndValues(unstructObj)...)
		}

	}
	return errCount
}

func (r *SyncConfigReconciler) deleteItems(rc *ReconciliationContext, targetNamespace corev1.Namespace) (errCount int) {
	for _, deleteItem := range rc.cfg.Spec.DeleteItems {
		r.Log.V(1).Info("Deleting", "item", deleteItem)
		deleteObj := deleteItem.ToDeleteObj(targetNamespace.Name)

		propagationPolicy := metav1.DeletePropagationBackground
		err := r.Client.Delete(rc.ctx, deleteObj, &client.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				errCount++
				r.Log.WithValues(getLoggingKeysAndValues(deleteObj)...).Info("Error deleting object", "error", err)
			}
		} else {
			r.Log.Info("Deleted", getLoggingKeysAndValues(deleteObj)...)
		}
	}
	return errCount
}

func (r *SyncConfigReconciler) getNamespaces(rc *ReconciliationContext) (namespaces []corev1.Namespace, returnErr error) {
	namespaceList := &corev1.NamespaceList{}
	err := r.Client.List(rc.ctx, namespaceList)
	if err != nil {
		return []corev1.Namespace{}, err
	}

	if rc.cfg.Spec.NamespaceSelector == nil {
		r.Log.Info("spec.namespaceSelector was not given, using namespace from SyncConfig", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
		namespaces = []corev1.Namespace{namespaceFromString(rc.cfg.Namespace)}
		return namespaces, nil
	}
	namespaces = filterNamespacesByNames(rc.cfg.Spec.NamespaceSelector.MatchNames, namespaceList.Items)

	labelSelector, err := metav1.LabelSelectorAsSelector(rc.cfg.Spec.NamespaceSelector.LabelSelector)
	if err != nil {
		return namespaces, err
	}

	for _, ns := range namespaceList.Items {
		if labelSelector.Matches(labels.Set(ns.GetLabels())) {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces, err
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
