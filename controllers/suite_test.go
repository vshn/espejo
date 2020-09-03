/*
Licensed under the Apache License, Version 2.0 (the "License");
http://www.apache.org/licenses/LICENSE-2.0
*/

package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
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

	err = syncv1alpha1.AddToScheme(scheme.Scheme)
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
		sc := &syncv1alpha1.SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ns},
			Spec: syncv1alpha1.SyncConfigSpec{
				SyncItems:         []unstructured.Unstructured{toUnstructured(cm)},
				NamespaceSelector: &syncv1alpha1.NamespaceSelector{MatchNames: []string{ns}},
			},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("reconciling sync config")
		result, err := syncConfigReconciler.Reconcile(ctrl.Request{
			NamespacedName: types.NamespacedName{Name: sc.Name, Namespace: sc.Namespace},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).ToNot(BeNil())

		By("verify that resource exist")
		syncResult := &v1.ConfigMap{}
		key := types.NamespacedName{Namespace: ns, Name: cm.Name}
		Expect(k8sClient.Get(context.Background(), key, syncResult)).ToNot(HaveOccurred())
		Expect(syncResult.Data["PROJECT_NAME"]).To(BeEquivalentTo(ns))
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
		sc := &syncv1alpha1.SyncConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test-syncconfig", Namespace: ns},
			Spec: syncv1alpha1.SyncConfigSpec{
				DeleteItems:       []syncv1alpha1.DeleteMeta{{Name: cm.Name, Kind: cm.Kind, APIVersion: cm.APIVersion}},
				NamespaceSelector: &syncv1alpha1.NamespaceSelector{MatchNames: []string{ns}},
			},
		}
		Expect(k8sClient.Create(context.Background(), sourceNs)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), cm)).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), sc)).ToNot(HaveOccurred())

		By("reconciling sync config")
		result, err := syncConfigReconciler.Reconcile(ctrl.Request{
			NamespacedName: types.NamespacedName{Name: sc.Name, Namespace: sc.Namespace},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).ToNot(BeNil())

		By("verify that resource doesn't exist anymore")
		err = k8sClient.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: cm.Name}, &v1.ConfigMap{})
		Expect(err).To(HaveOccurred())
	})
})

func toUnstructured(configMap *v1.ConfigMap) unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(configMap)
	Expect(err).ToNot(HaveOccurred())
	obj.SetUnstructuredContent(m)
	return obj
}
