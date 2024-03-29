//go:build integration

package controllers

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// +kubebuilder:scaffold:imports

	"github.com/vshn/espejo/api/v1alpha1"
)

type EnvTestSuite struct {
	suite.Suite

	NS     string
	Client client.Client
	Config *rest.Config
	Env    *envtest.Environment
	Logger logr.Logger
	Ctx    context.Context
	Scheme *runtime.Scheme
}

func (ts *EnvTestSuite) SetupSuite() {
	ts.Logger = zapr.NewLogger(zaptest.NewLogger(ts.T()))
	log.SetLogger(ts.Logger)

	ts.Ctx = context.Background()

	testEnv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "apiextensions.k8s.io", "v1", "base")},
	}

	config, err := testEnv.Start()
	ts.Require().NoError(err)
	ts.Require().NotNil(config)

	registerCRDs(ts)

	k8sClient, err := client.New(config, client.Options{
		Scheme: ts.Scheme,
	})
	ts.Require().NoError(err)
	ts.Require().NotNil(k8sClient)

	ts.Env = testEnv
	ts.Config = config
	ts.Client = k8sClient
}

func registerCRDs(ts *EnvTestSuite) {
	ts.Scheme = runtime.NewScheme()
	ts.Require().NoError(scheme.AddToScheme(ts.Scheme))
	ts.Require().NoError(metav1.AddMetaToScheme(ts.Scheme))
	ts.Require().NoError(corev1.AddToScheme(ts.Scheme))
	ts.Require().NoError(v1alpha1.AddToScheme(ts.Scheme))

	// +kubebuilder:scaffold:scheme
}

func (ts *EnvTestSuite) TearDownSuite() {
	err := ts.Env.Stop()
	ts.Require().NoError(err)
}

// NewNS instantiates a new Namespace object with the given name.
func (ts *EnvTestSuite) NewNS(nsName string) *corev1.Namespace {
	ts.Assert().Emptyf(validation.IsDNS1123Label(nsName), "'%s' does not appear to be a valid name for a namespace", nsName)

	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
}

// EnsureNS creates a new Namespace object using EnvTestSuite.Client.
func (ts *EnvTestSuite) EnsureNS(nsName string) {
	ns := ts.NewNS(nsName)
	ts.T().Logf("creating namespace '%s'", nsName)
	ts.Require().NoError(ts.Client.Create(ts.Ctx, ns))
}

// EnsureResources ensures that the given resources are existing in the suite. Each error will fail the test.
func (ts *EnvTestSuite) EnsureResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("creating resource '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Create(ts.Ctx, resource))
	}
}

// UpdateResources ensures that the given resources are updated in the suite. Each error will fail the test.
func (ts *EnvTestSuite) UpdateResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("updating resource '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Update(ts.Ctx, resource))
	}
}

// UpdateStatus ensures that the Status property of the given resources are updated in the suite. Each error will fail the test.
func (ts *EnvTestSuite) UpdateStatus(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("updating status '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Status().Update(ts.Ctx, resource))
	}
}

// DeleteResources deletes the given resources are updated from the suite. Each error will fail the test.
func (ts *EnvTestSuite) DeleteResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("deleting '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Delete(ts.Ctx, resource))
	}
}

// FetchResource fetches the given object name and stores the result in the given object.
// Test fails on errors.
func (ts *EnvTestSuite) FetchResource(name types.NamespacedName, object client.Object) {
	ts.Require().NoError(ts.Client.Get(ts.Ctx, name, object))
}

// FetchResource fetches resources and puts the items into the given list with the given list options.
// Test fails on errors.
func (ts *EnvTestSuite) FetchResources(objectList client.ObjectList, opts ...client.ListOption) {
	ts.Require().NoError(ts.Client.List(ts.Ctx, objectList, opts...))
}

// MapToRequest maps the given object into a reconcile Request.
func (ts *EnvTestSuite) MapToRequest(object metav1.Object) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      object.GetName(),
			Namespace: object.GetNamespace(),
		},
	}
}

// BeforeTest is invoked just before every test starts
func (ts *EnvTestSuite) SetupTest() {
	ts.NS = rand.String(8)
	ts.EnsureNS(ts.NS)
}

// IsResourceExisting tries to fetch the given resource and returns true if it exists.
// It will consider still-existing object with a deletion timestamp as not existing.
// Any other errors will fail the test.
func (ts *EnvTestSuite) IsResourceExisting(ctx context.Context, obj client.Object) bool {
	err := ts.Client.Get(ctx, ts.MapToNamespacedName(obj), obj)
	if apierrors.IsNotFound(err) {
		return false
	}
	ts.Assert().NoError(err)
	return obj.GetDeletionTimestamp() == nil
}

// MapToNamespacedName returns the NamespacedName for the given object.
func (ts *EnvTestSuite) MapToNamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}
