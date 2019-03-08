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
	ctx "context"
	"testing"

	"github.com/vshn/espejo/pkg/apis/sync/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestProcessTemplate(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAnnotations(map[string]string{"replace": "${PROJECT_NAME}"})
	obj.SetNamespace("name-with-suffix-${PROJECT_NAME}")

	processTemplate(obj, "test-name")

	if obj.GetNamespace() != "name-with-suffix-test-name" {
		t.Errorf("Name was not replaced: '%v'", obj.GetNamespace())
	}
	test, ok := obj.GetAnnotations()["replace"]
	if !ok || test != "test-name" {
		t.Errorf("Annotation was not replaced: '%v'", test)
	}
}

func TestGetNamespaces(t *testing.T) {
	testNamespaces := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace-one",
				Labels: map[string]string{"a": "b"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace-two",
				Labels: map[string]string{"a": "b"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace-three",
				Labels: map[string]string{"a": "c"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace-four",
			},
		},
	}
	testTable := []struct {
		name       string
		namespaces []runtime.Object
		syncConfig *v1alpha1.SyncConfig
		matchCount int
	}{
		{
			"Match single name",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						MatchNames: []string{"test-namespace-one"},
					},
				},
			},
			1,
		},
		{
			"Match Exists expression",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								metav1.LabelSelectorRequirement{Key: "a", Operator: metav1.LabelSelectorOpExists},
							},
						},
					},
				},
			},
			3,
		},
		{
			"Match In expression",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								metav1.LabelSelectorRequirement{Key: "a", Operator: metav1.LabelSelectorOpIn, Values: []string{"b"}},
							},
						},
					},
				},
			},
			2,
		},
		{
			"Match NotIn expression",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								metav1.LabelSelectorRequirement{Key: "a", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"b"}},
							},
						},
					},
				},
			},
			2,
		},
		{
			"Match label expression",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"a": "c"},
						},
					},
				},
			},
			1,
		},
		{
			"Match all",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
					},
				},
			},
			len(testNamespaces),
		},
		{
			"Match name and label",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{
						MatchNames: []string{"test-namespace-four"},
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"a": "b"},
						},
					},
				},
			},
			3,
		},
		{
			"Match none",
			testNamespaces,
			&v1alpha1.SyncConfig{
				Spec: v1alpha1.SyncConfigSpec{
					NamespaceSelector: &v1alpha1.NamespaceSelector{},
				},
			},
			0,
		},
	}
	r := &ReconcileSyncConfig{}
	request := reconcile.Request{}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			r.client = fake.NewFakeClient(testCase.namespaces...)
			namespaces, err := r.getNamespaces(ctx.TODO(), testCase.syncConfig, request)
			if err != nil {
				t.Error(err)
			}
			if len(namespaces) != testCase.matchCount {
				t.Errorf("Found %d namespaces, expected %d", len(namespaces), testCase.matchCount)
			}
		})
	}
}
