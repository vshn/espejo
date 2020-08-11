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
	"strings"
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
		client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
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
		Complete(r)
}

// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sync.appuio.ch,resources=syncconfigs/status,verbs=get;update;patch

func (r *SyncConfigReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, returnErr error) {
	ctx := context.Background()
	_ = r.Log.WithValues("syncconfig", req.NamespacedName)

	syncConfig := &syncv1alpha1.SyncConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, syncConfig)
	if err != nil {
		r.Log.Error(err, "could not get sync config", "syncconfig", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	rc := &ReconciliationContext{
		ctx: ctx,
		cfg: syncConfig,
	}
	errCount := 0
	namespaces, err := r.getNamespaces(rc)

	for _, targetNamespace := range namespaces {
		errCount += r.deleteItems(rc, targetNamespace)
		errCount += r.syncItems(rc, targetNamespace)
	}
	if errCount > 0 {
		returnErr = fmt.Errorf("%v errors occured during reconcile loop", errCount)
	}
	_ = r.Client.Update(rc.ctx, syncConfig)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, returnErr
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
		deleteObj := deleteItem.ToDeleteObj(targetNamespace.Name)

		propagationPolicy := metav1.DeletePropagationOrphan
		err := r.Client.Delete(rc.ctx, deleteObj, &client.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				errCount++
				r.Log.Error(err, "Error deleting object", getLoggingKeysAndValues(deleteObj)...)
			}
		} else {
			r.Log.Info("Deleted", getLoggingKeysAndValues(deleteObj)...)
		}
	}
	return errCount
}

func (r *SyncConfigReconciler) getNamespaces(rc *ReconciliationContext) ([]corev1.Namespace, error) {
	namespaceList := &corev1.NamespaceList{}
	err := r.Client.List(rc.ctx, namespaceList)
	if err != nil {
		return []corev1.Namespace{}, err
	}

	namespaces := filterNamespacesByNames(rc.cfg.Spec.NamespaceSelector.MatchNames, namespaceList.Items)

	selector, err := metav1.LabelSelectorAsSelector(rc.cfg.Spec.NamespaceSelector.LabelSelector)
	if err != nil {
		return namespaces, err
	}

	for _, ns := range namespaceList.Items {
		if selector.Matches(labels.Set(ns.GetLabels())) {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces, err
}

func filterNamespacesByNames(names []string, namespaceList []corev1.Namespace) (namespaces []corev1.Namespace) {
	nameLookup := make(map[string]bool, len(names))

	for _, name := range names {
		nameLookup[name] = true
	}

	for _, ns := range namespaceList {
		if _, found := nameLookup[ns.Name]; found {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces
}

func replaceProjectName(replacement string, m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			continue
		}
		switch v.(type) {
		case string:
			s := m[k].(string)
			m[k] = strings.ReplaceAll(s, "${PROJECT_NAME}", replacement)
		case interface{}:
			replaceProjectName(replacement, m[k].(map[string]interface{}))
		}
	}
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
