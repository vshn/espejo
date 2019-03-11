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
	"strings"

	templateapi "github.com/openshift/origin/pkg/template/apis/template"
	"github.com/openshift/origin/pkg/template/generator"
	"github.com/openshift/origin/pkg/template/templateprocessing"
	"github.com/vshn/espejo/pkg/apis/sync/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func recreateOject(ctx context.Context, unstructObj *unstructured.Unstructured, c client.Client) error {
	unstructObj.SetResourceVersion("")

	err := c.Delete(ctx, unstructObj, client.DeleteOptionFunc(func(d *client.DeleteOptions) {
		option := metav1.DeletePropagationForeground
		d.PropagationPolicy = &option
	}))

	if err != nil {
		return err
	}

	err = c.Create(context.Background(), unstructObj)
	if err != nil {
		return err
	}

	return nil
}

func processTemplate(unstructObj *unstructured.Unstructured, projectName string) error {
	params := make(map[string]templateapi.Parameter, 1)
	params["PROJECT_NAME"] = templateapi.Parameter{
		Name:  "PROJECT_NAME",
		Value: projectName,
	}

	generatorMap := make(map[string]generator.Generator)
	processor := templateprocessing.NewProcessor(generatorMap)

	_, err := processor.SubstituteParameters(params, unstructObj)

	return err
}

func getLoggingKeysAndValues(unstructuredObject *unstructured.Unstructured) []interface{} {
	return []interface{}{
		"Object.Kind", unstructuredObject.GetKind(),
		"Object.Namespace", unstructuredObject.GetNamespace(),
		"Object.Name", unstructuredObject.GetName(),
	}
}

func (r *ReconcileSyncConfig) getNamespaces(ctx context.Context, syncConfig *v1alpha1.SyncConfig, request reconcile.Request) ([]corev1.Namespace, error) {
	namespaceList := &corev1.NamespaceList{}
	err := r.client.List(ctx, &client.ListOptions{
		Namespace: "",
		Raw: &metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
		},
	}, namespaceList)
	if err != nil {
		return []corev1.Namespace{}, err
	}

	namespaces := filterNamespacesByNames(syncConfig.Spec.NamespaceSelector.MatchNames, namespaceList.Items)

	selector, err := convertSelector(syncConfig.Spec.NamespaceSelector.LabelSelector)
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

func filterNamespacesByNames(names []string, namespaceList []corev1.Namespace) []corev1.Namespace {
	namespaces := []corev1.Namespace{}
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

func convertSelector(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector := labels.SelectorFromSet(labels.Set(labelSelector.MatchLabels))
		for _, req := range labelSelector.MatchExpressions {
			op := strings.ToLower(string(req.Operator))
			r, err := labels.NewRequirement(req.Key, selection.Operator(op), req.Values)
			if err != nil {
				return nil, err
			}
			selector = selector.Add(*r)
		}
		return selector, nil
	}
	return labels.Nothing(), nil
}
