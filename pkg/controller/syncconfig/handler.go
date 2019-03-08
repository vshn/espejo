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
	"strings"

	"github.com/vshn/espejo/pkg/apis/sync/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileSyncConfig) handle(ctx context.Context, syncConfig *v1alpha1.SyncConfig, request reconcile.Request) error {
	namespaces, err := r.getNamespaces(ctx, syncConfig, request)
	if err != nil {
		log.Error(err, "Error getting namespaces")
		return err
	}
	var errors []error
	for _, targetNamespace := range namespaces {
		for _, deleteItem := range syncConfig.Spec.DeleteItems {
			deleteObj := &unstructured.Unstructured{}
			deleteObj.SetAPIVersion(deleteItem.APIVersion)
			deleteObj.SetKind(deleteItem.Kind)
			deleteObj.SetName(deleteItem.Name)
			deleteObj.SetNamespace(targetNamespace.Name)

			err = r.client.Delete(ctx, deleteObj, client.DeleteOptionFunc(func(d *client.DeleteOptions) {
				option := metav1.DeletePropagationBackground
				d.PropagationPolicy = &option
			}))
			if err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "Error deleting object", getLoggingKeysAndValues(deleteObj)...)
					errors = append(errors, err)
				}
			} else {
				log.Info("Deleted", getLoggingKeysAndValues(deleteObj)...)
			}
		}
		for _, item := range syncConfig.Spec.Items {
			unstructObj := item.DeepCopy()
			unstructObj.SetNamespace(targetNamespace.Name)

			err := processTemplate(unstructObj, targetNamespace.Name)
			if err != nil {
				log.Error(err, "Error processing template", getLoggingKeysAndValues(unstructObj)...)
				errors = append(errors, err)
				continue
			}

			err = r.client.Create(ctx, unstructObj)
			if err != nil {
				if !apierrors.IsAlreadyExists(err) {
					log.Error(err, "Error creating", getLoggingKeysAndValues(unstructObj)...)
					errors = append(errors, err)
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
					errors = append(errors, err)
					continue
				}
				unstructObj.SetResourceVersion(existingObj.GetResourceVersion())
				err = r.client.Update(ctx, unstructObj)
				if err != nil {
					if apierrors.IsInvalid(err) && syncConfig.Spec.ForceRecreate {
						log.Info("Recreating", getLoggingKeysAndValues(unstructObj)...)
						err = recreateOject(ctx, unstructObj, r.client)
						if err != nil {
							log.Error(err, "Error recreating", getLoggingKeysAndValues(unstructObj)...)
							errors = append(errors, err)
						}
					} else {
						log.Error(err, "Error updating", getLoggingKeysAndValues(unstructObj)...)
						errors = append(errors, err)
					}
				} else {
					log.Info("Updated", getLoggingKeysAndValues(unstructObj)...)
				}
			} else {
				log.Info("Created", getLoggingKeysAndValues(unstructObj)...)
			}
		}
	}
	if len(errors) > 0 {
		errorStrings := make([]string, len(errors))
		for _, e := range errors {
			errorStrings = append(errorStrings, e.Error())
		}
		return fmt.Errorf(strings.Join(errorStrings, "\n"))
	}

	return nil
}
