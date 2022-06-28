//go:build integration

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/vshn/espejo/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

type SyncConfigControllerTestSuite struct {
	EnvTestSuite
	reconciler *SyncConfigReconciler
}

func Test_SyncConfig(t *testing.T) {
	suite.Run(t, new(SyncConfigControllerTestSuite))
}

func (ts *SyncConfigControllerTestSuite) BeforeTest(suiteName, testName string) {
	ts.reconciler = &SyncConfigReconciler{
		Client: ts.Client,
		Log:    ts.Logger.WithName(suiteName + "_" + testName),
		Scheme: ts.Scheme,
	}
}

func (ts *SyncConfigControllerTestSuite) Test_GivenNewSyncConfig_WhenReconcile_ThenCreateNewResources() {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-configmap",
		},
		Data: map[string]string{"PROJECT_NAME": "${PROJECT_NAME}"},
	}
	sc := &SyncConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ts.NS},
		Spec: SyncConfigSpec{
			SyncItems:         []unstructured.Unstructured{toUnstructured(ts.T(), cm)},
			NamespaceSelector: &NamespaceSelector{MatchNames: []string{ts.NS}},
		},
	}
	ts.EnsureResources(sc)
	result, err := ts.reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: ts.MapToNamespacedName(sc),
	})
	ts.Require().NoError(err)
	ts.Assert().NotNil(result)

	cm.Namespace = ts.NS
	ts.FetchResource(ts.MapToNamespacedName(cm), cm)
	ts.Assert().Equal(ts.NS, cm.Data["PROJECT_NAME"])

	ts.FetchResource(ts.MapToNamespacedName(sc), sc)
	ts.Assert().Equal(int64(0), sc.Status.DeletedItemCount)
	ts.Assert().Equal(int64(0), sc.Status.FailedItemCount)
	ts.Assert().Equal(int64(1), sc.Status.SynchronizedItemCount)
}

func (ts *SyncConfigControllerTestSuite) Test_GivenSyncConfig_WhenReconcile_ThenUpdateResources() {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-configmap",
		},
		Data: map[string]string{"PROJECT_NAME": "${PROJECT_NAME}"},
	}
	sc := &SyncConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ts.NS},
		Spec: SyncConfigSpec{
			SyncItems:         []unstructured.Unstructured{toUnstructured(ts.T(), cm)},
			NamespaceSelector: &NamespaceSelector{MatchNames: []string{ts.NS}},
		},
	}
	cm.Namespace = ts.NS
	cm.Data["PROJECT_NAME"] = "wrong"
	cm.Data["other"] = "new"
	ts.EnsureResources(cm, sc)
	result, err := ts.reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: ts.MapToNamespacedName(sc),
	})
	ts.Require().NoError(err)
	ts.Assert().NotNil(result)

	ts.FetchResource(ts.MapToNamespacedName(cm), cm)
	ts.Assert().Equal(ts.NS, cm.Data["PROJECT_NAME"])
	ts.Assert().NotContains(cm.Data, "other")

	ts.FetchResource(ts.MapToNamespacedName(sc), sc)
	ts.Assert().Equal(int64(0), sc.Status.DeletedItemCount)
	ts.Assert().Equal(int64(0), sc.Status.FailedItemCount)
	ts.Assert().Equal(int64(1), sc.Status.SynchronizedItemCount)
}

type invalidUpdateClient struct {
	client.Client
}

func (c invalidUpdateClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return apierrors.NewInvalid(obj.GetObjectKind().GroupVersionKind().GroupKind(), obj.GetName(), nil)
}
func (c invalidUpdateClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
  // Envtest cannot handle foreground deletions used in force recreate as no GC controller is running, so we drop options
  return c.Client.Delete(ctx, obj)
}

func (ts *SyncConfigControllerTestSuite) Test_GivenForceSyncConfig_WhenReconcile_ThenForceUpdateResources() {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-configmap",
		},
		Data: map[string]string{"PROJECT_NAME": "${PROJECT_NAME}"},
	}
	sc := &SyncConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ts.NS},
		Spec: SyncConfigSpec{
			SyncItems:         []unstructured.Unstructured{toUnstructured(ts.T(), cm)},
			NamespaceSelector: &NamespaceSelector{MatchNames: []string{ts.NS}},
			ForceRecreate:     true,
		},
	}
	cm.Namespace = ts.NS
	cm.Data["PROJECT_NAME"] = "wrong"
	cm.Data["other"] = "new"
	ts.EnsureResources(cm, sc)
	reconciler := &SyncConfigReconciler{
		Client: invalidUpdateClient{ts.reconciler.Client},
		Log:    ts.reconciler.Log,
		Scheme: ts.reconciler.Scheme,
	}

	result, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: ts.MapToNamespacedName(sc),
	})
	ts.Require().NoError(err)
	ts.Assert().NotNil(result)

	ts.FetchResource(ts.MapToNamespacedName(cm), cm)
	ts.Assert().Equal(ts.NS, cm.Data["PROJECT_NAME"])
	ts.Assert().NotContains(cm.Data, "other")

	ts.FetchResource(ts.MapToNamespacedName(sc), sc)
	ts.Assert().Equal(int64(0), sc.Status.DeletedItemCount)
	ts.Assert().Equal(int64(0), sc.Status.FailedItemCount)
	ts.Assert().Equal(int64(1), sc.Status.SynchronizedItemCount)
}

func (ts *SyncConfigControllerTestSuite) Test_GivenSyncConfigWithDelete_WhenReconcile_ThenDeleteResource() {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: ts.NS,
		},
	}
	sc := &SyncConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ts.NS},
		Spec: SyncConfigSpec{
			DeleteItems:       []DeleteMeta{{Name: cm.Name, Kind: cm.Kind, APIVersion: cm.APIVersion}},
			NamespaceSelector: &NamespaceSelector{MatchNames: []string{ts.NS}},
		},
	}
	ts.EnsureResources(cm, sc)
	result, err := ts.reconciler.Reconcile(ts.Ctx, ctrl.Request{
		NamespacedName: ts.MapToNamespacedName(sc),
	})
	ts.Require().NoError(err)
	ts.Assert().NotNil(result)

	ts.Assert().False(ts.IsResourceExisting(ts.Ctx, cm))
	ts.FetchResource(ts.MapToNamespacedName(sc), sc)

	ts.reconciler.Log.Info("status", "status", sc.Status)
	ts.Assert().Equal(int64(1), sc.Status.DeletedItemCount)
	ts.Assert().Equal(int64(0), sc.Status.FailedItemCount)
	ts.Assert().Equal(int64(0), sc.Status.SynchronizedItemCount)
}

func (ts *SyncConfigControllerTestSuite) Test_GivenInvalidConfig_WhenReconcile_ThenAbortAndUpdateStatus() {
	sc := &SyncConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ts.NS},
		Spec:       SyncConfigSpec{},
	}
	ts.EnsureResources(sc)
	result, err := ts.reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: ts.MapToNamespacedName(sc),
	})
	ts.Require().NotNil(result)
	ts.Require().NoError(err)
	ts.Assert().False(result.Requeue)

	ts.FetchResource(ts.MapToNamespacedName(sc), sc)
	condition := meta.FindStatusCondition(sc.Status.Conditions, ConditionInvalid.String())
	ts.Assert().NotNil(condition)
}
