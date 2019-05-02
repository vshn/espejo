// === Authors
//
// Simon Rüegg <simon.ruegg@vshn.ch>
//
// === License
//
// Copyright (c) 2019, VSHN AG, info@vshn.ch
// Licensed under "BSD 3-Clause". See LICENSE file.
//

package syncconfig

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/vshn/espejo/pkg/apis/sync/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileSyncConfig) handle(ctx context.Context, syncConfig *v1alpha1.SyncConfig, request reconcile.Request, reqLogger logr.Logger) error {
	errCount := 0
	namespaces, err := r.getNamespaces(ctx, syncConfig, request)
	if err != nil {
		errCount++
		reqLogger.Error(err, "Error getting namespaces")
	}

	for _, targetNamespace := range namespaces {
		for _, deleteItem := range syncConfig.Spec.DeleteItems {
			deleteObj := deleteItem.ToDeleteObj(targetNamespace.Name)

			err = r.client.Delete(ctx, deleteObj, client.DeleteOptionFunc(func(d *client.DeleteOptions) {
				option := metav1.DeletePropagationOrphan
				d.PropagationPolicy = &option
			}))
			if err != nil {
				if !apierrors.IsNotFound(err) {
					errCount++
					reqLogger.Error(err, "Error deleting object", getLoggingKeysAndValues(deleteObj)...)
				}
			} else {
				reqLogger.Info("Deleted", getLoggingKeysAndValues(deleteObj)...)
			}
		}
		for _, item := range syncConfig.Spec.SyncItems {
			unstructObj := item.DeepCopy()
			unstructObj.SetNamespace(targetNamespace.Name)

			err := processTemplate(unstructObj, targetNamespace.Name)
			if err != nil {
				errCount++
				reqLogger.Error(err, "Error processing template", getLoggingKeysAndValues(unstructObj)...)
				continue
			}

			err = r.client.Create(ctx, unstructObj)
			if err != nil {
				if !apierrors.IsAlreadyExists(err) {
					errCount++
					reqLogger.Error(err, "Error creating", getLoggingKeysAndValues(unstructObj)...)
					continue
				}
				existingObj := &unstructured.Unstructured{}
				existingObj.SetKind(unstructObj.GetKind())
				existingObj.SetAPIVersion(unstructObj.GetAPIVersion())
				err = r.client.Get(ctx,
					types.NamespacedName{
						Name:      unstructObj.GetName(),
						Namespace: unstructObj.GetNamespace()},
					existingObj)
				if err != nil {
					errCount++
					continue
				}
				unstructObj.SetResourceVersion(existingObj.GetResourceVersion())
				err = r.client.Update(ctx, unstructObj)
				if err != nil {
					if apierrors.IsInvalid(err) && syncConfig.Spec.ForceRecreate {
						reqLogger.Info("Recreating", getLoggingKeysAndValues(unstructObj)...)
						err = recreateOject(ctx, unstructObj, r.client)
						if err != nil {
							errCount++
							reqLogger.Error(err, "Error recreating", getLoggingKeysAndValues(unstructObj)...)
						}
					} else {
						errCount++
						reqLogger.Error(err, "Error updating", getLoggingKeysAndValues(unstructObj)...)
					}
				} else {
					reqLogger.Info("Updated", getLoggingKeysAndValues(unstructObj)...)
				}
			} else {
				reqLogger.Info("Created", getLoggingKeysAndValues(unstructObj)...)
			}
		}
	}
	if errCount > 0 {
		return fmt.Errorf("%v errors occured during reconcile loop", errCount)
	}

	return nil
}
