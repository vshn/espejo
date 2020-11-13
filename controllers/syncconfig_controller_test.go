package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vshn/espejo/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                  *rest.Config
	k8sClient            client.Client
	testEnv              *envtest.Environment
	syncConfigReconciler *SyncConfigReconciler
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	By("setting up the controller")
	syncConfigReconciler = &SyncConfigReconciler{Client: k8sClient, Log: logf.Log, Scheme: scheme.Scheme}

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("SyncConfig controller", func() {

	It("should create new resources from SyncConfig", func() {

		By("setting up test resources")
		ns := "create-resource"
		sourceNs := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		cm := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-configmap",
			},
			Data: map[string]string{"PROJECT_NAME": "${PROJECT_NAME}"},
		}
		sc := &SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ns},
			Spec: SyncConfigSpec{
				SyncItems:         []unstructured.Unstructured{toUnstructured(cm)},
				NamespaceSelector: &NamespaceSelector{MatchNames: []string{ns}},
			},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("reconciling sync config")
		result, err := syncConfigReconciler.Reconcile(ctrl.Request{
			NamespacedName: toObjectKey(sc),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).ToNot(BeNil())

		By("verify that resource exist")
		cmResult := &v1.ConfigMap{}
		cmKey := types.NamespacedName{Namespace: ns, Name: cm.Name}
		Expect(k8sClient.Get(context.Background(), cmKey, cmResult)).ToNot(HaveOccurred())
		Expect(cmResult.Data["PROJECT_NAME"]).To(BeEquivalentTo(ns))

		newSC := &SyncConfig{ObjectMeta: toObjectMeta(sc.Name, sc.Namespace)}
		err = k8sClient.Get(context.Background(), toObjectKey(newSC), newSC)
		Expect(err).ToNot(HaveOccurred())
		Expect(newSC.Status.DeletedItemCount).To(Equal(int64(0)))
		Expect(newSC.Status.FailedItemCount).To(Equal(int64(0)))
		Expect(newSC.Status.SynchronizedItemCount).To(Equal(int64(1)))
	})

	It("should delete existing resources from SyncConfig", func() {

		By("setting up test resources")
		ns := "delete-resource"
		sourceNs := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		cm := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: ns,
			},
		}
		sc := &SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ns},
			Spec: SyncConfigSpec{
				DeleteItems:       []DeleteMeta{{Name: cm.Name, Kind: cm.Kind, APIVersion: cm.APIVersion}},
				NamespaceSelector: &NamespaceSelector{MatchNames: []string{ns}},
			},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), cm)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("reconciling sync config")
		result, err := syncConfigReconciler.Reconcile(ctrl.Request{
			NamespacedName: toObjectKey(sc),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).ToNot(BeNil())

		By("verify that resource doesn't exist anymore")
		err = k8sClient.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: cm.Name}, &v1.ConfigMap{})
		Expect(err).To(HaveOccurred())

		newSC := &SyncConfig{ObjectMeta: toObjectMeta(sc.Name, sc.Namespace)}
		err = k8sClient.Get(context.Background(), toObjectKey(newSC), newSC)
		Expect(err).ToNot(HaveOccurred())
		syncConfigReconciler.Log.Info("status", "status", newSC.Status)
		Expect(newSC.Status.DeletedItemCount).To(Equal(int64(1)))
		Expect(newSC.Status.FailedItemCount).To(Equal(int64(0)))
		Expect(newSC.Status.SynchronizedItemCount).To(Equal(int64(0)))
	})

	It("should map namespace updates into list of reconcile objects", func() {

		By("setting up test resources")
		ns := "map"
		sourceNs := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		sc := &SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: sourceNs.Name},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("mapping namespace into reconcile object")
		result := syncConfigReconciler.Map(handler.MapObject{Meta: sourceNs.GetObjectMeta()})
		Expect(result).ToNot(BeEmpty())
		Expect(result).To(ContainElement(reconcile.Request{NamespacedName: types.NamespacedName{Name: sc.Name, Namespace: sc.Namespace}}))
	})

	It("should map namespace updates into list of reconcile objects with namespace filter", func() {

		By("setting up test resources")
		ns := "map-filtered"
		sourceNs := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		sc := &SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: sourceNs.Name},
		}
		syncConfigReconciler.WatchNamespace = ns
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("mapping namespace into reconcile object")
		result := syncConfigReconciler.Map(handler.MapObject{Meta: sourceNs.GetObjectMeta()})
		Expect(result).To(HaveLen(1))
		Expect(result).To(ContainElement(reconcile.Request{NamespacedName: types.NamespacedName{Name: sc.Name, Namespace: sc.Namespace}}))
		syncConfigReconciler.WatchNamespace = ""
	})

	It("should not reconcile SyncConfig upon failed validation", func() {

		By("setting up test resources")
		ns := "validation-fail"
		sourceNs := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		sc := &SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: sourceNs.Name},
			Spec:       SyncConfigSpec{},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("reconciling sync config")
		result, err := syncConfigReconciler.Reconcile(ctrl.Request{
			NamespacedName: toObjectKey(sc),
		})
		Expect(result.Requeue).To(BeFalse())

		By("verifying sync config status")
		newSC := &SyncConfig{ObjectMeta: toObjectMeta(sc.Name, sc.Namespace)}
		err = k8sClient.Get(context.Background(), toObjectKey(newSC), newSC)
		Expect(err).ToNot(HaveOccurred())
		conditions := mapConditionsToType(newSC.Status.Conditions)
		Expect(conditions).To(HaveKey(SyncConfigInvalid))
	})
})

func toUnstructured(obj interface{}) unstructured.Unstructured {
	converted := unstructured.Unstructured{}
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	Expect(err).ToNot(HaveOccurred())
	converted.SetUnstructuredContent(m)
	return converted
}

func toObjectKey(config *SyncConfig) types.NamespacedName {
	return types.NamespacedName{Name: config.Name, Namespace: config.Namespace}
}

func toObjectMeta(name, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}
