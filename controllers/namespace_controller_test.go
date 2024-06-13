//go:build integration

package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/vshn/espejo/api/v1alpha1"
	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

type NamespaceControllerTestSuite struct {
	EnvTestSuite
	scopedNs   string
	reconciler *NamespaceReconciler
}

func Test_Namespace(t *testing.T) {
	suite.Run(t, new(NamespaceControllerTestSuite))
}

func (ts *NamespaceControllerTestSuite) BeforeTest(suiteName, testName string) {
	ts.scopedNs = "scoped-" + rand.String(5)
	ts.reconciler = &NamespaceReconciler{
		Client:                  ts.Client,
		Log:                     ts.Logger.WithName(suiteName + "_" + testName),
		Scheme:                  ts.Scheme,
		NewSyncConfigReconciler: ts.newSupplier(ts.scopedNs),
	}
	ns := namespaceFromString(ts.scopedNs)
	ts.EnsureResources(&ns)
}

func (ts *NamespaceControllerTestSuite) Test_GivenNamespaceReconciler_WhenNamespaceUpdates_ThenLimitSyncToNamespaceOnly() {
	templateCm, _ := ts.givenSyncConfig("*")

	ts.whenReconciling()

	ts.thenAssertSyncHappenedOnlyInScopedNamespace(templateCm)
}

func (ts *NamespaceControllerTestSuite) Test_GivenNamespaceReconciler_WhenAnIgnoredNamespaceUpdates_ThenDontSyncThatNamespace() {
	templateCm, _ := ts.givenSyncConfig("shouldn't match")

	ts.whenReconciling()

	ts.thenAssertResourceDoesNotExist(templateCm)
}

func (ts *NamespaceControllerTestSuite) whenReconciling() {
	result, err := ts.reconciler.Reconcile(ts.Ctx, ts.mapToNamespaceRequest(ts.scopedNs))

	ts.Assert().NoError(err)
	ts.Assert().False(result.Requeue)
	ts.Assert().Equal(time.Duration(0), result.RequeueAfter)
}

func (ts *NamespaceControllerTestSuite) newSupplier(scopedNS string) func() *SyncConfigReconciler {
	return func() *SyncConfigReconciler {
		return &SyncConfigReconciler{
			NamespaceScope: scopedNS,
			Log:            ts.Logger.WithName(scopedNS),
			Scheme:         ts.Scheme,
			Client:         ts.Client,
		}
	}
}

func (ts *NamespaceControllerTestSuite) mapToNamespaceRequest(namespace string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{Name: namespace},
	}
}

func (ts *NamespaceControllerTestSuite) givenSyncConfig(matchNames string) (*corev1.ConfigMap, *SyncConfig) {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "scoped-configmap-" + rand.String(5),
		},
	}
	sc := &SyncConfig{
		ObjectMeta: toObjectMeta("test-syncconfig", ts.NS),
		Spec: SyncConfigSpec{
			SyncItems:         []syncv1alpha1.Manifest{syncv1alpha1.Manifest{toUnstructured(ts.T(), cm)}},
			NamespaceSelector: &NamespaceSelector{MatchNames: []string{matchNames}},
		},
	}
	ts.EnsureResources(sc)
	return cm, sc
}

func (ts *NamespaceControllerTestSuite) thenAssertSyncHappenedOnlyInScopedNamespace(templateCm *corev1.ConfigMap) {
	cmList := &corev1.ConfigMapList{}
	ts.FetchResources(cmList)
	ts.Assert().NotEmpty(cmList.Items)
	count := 0
	for _, cm := range cmList.Items {
		if cm.Name == templateCm.Name {
			count++
			ts.Assert().Equal(cm.Namespace, ts.scopedNs)
		}
	}
	ts.Assert().Equal(1, count)
}

func (ts *NamespaceControllerTestSuite) thenAssertResourceDoesNotExist(cm *corev1.ConfigMap) {
	nonExistingCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.Name,
			Namespace: ts.scopedNs,
		},
	}
	ts.Assert().False(ts.IsResourceExisting(ts.Ctx, nonExistingCm))
}
