package main

import (
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/posflag"
	flag "github.com/spf13/pflag"
	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
	"github.com/vshn/espejo/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	// These will be populated by Goreleaser
	version string
	commit  string
	date    string

	scheme        = runtime.NewScheme()
	setupLog      = ctrl.Log.WithName("setup")
	koanfInstance = koanf.New(".")
	config        = Configuration{
		LeaderElection:    false,
		MetricsAddr:       ":8080",
		ReconcileInterval: "10s",
	}
)

type (
	Configuration struct {
		LeaderElection    bool   `koanf:"enable-leader-election"`
		MetricsAddr       string `koanf:"metrics-addr"`
		ReconcileInterval string `koanf:"reconcile-interval"`
	}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(syncv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.String("metrics-addr", config.MetricsAddr, "The address the metric endpoint binds to.")
	f.Bool("enable-leader-election", config.LeaderElection, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	f.String("reconcile-interval", config.ReconcileInterval, "The interval of which SyncConfigs get reconciled.")

	loadConfig(f)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: config.MetricsAddr,
		Port:               9443,
		LeaderElection:     config.LeaderElection,
		LeaderElectionID:   "bd39f6a0.appuio.ch",
		Namespace:          getWatchNamespace(),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	interval, err := time.ParseDuration(config.ReconcileInterval)
	if err != nil {
		setupLog.Error(err, "could not parse interval")
		os.Exit(1)
	}
	if err = (&controllers.SyncConfigReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("SyncConfig"),
		Scheme:            mgr.GetScheme(),
		ReconcileInterval: interval,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SyncConfig")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.WithValues("version", version, "date", date, "commit", commit).Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
// An empty value means the operator is running with cluster scope.
func getWatchNamespace() string {
	return os.Getenv("WATCH_NAMESPACE")
}

// loadConfig will populate the configuration
func loadConfig(f *flag.FlagSet) {
	_ = f.Parse(os.Args[1:])
	if err := koanfInstance.Load(posflag.Provider(f, ".", koanfInstance), nil); err != nil {
		setupLog.Error(err, "error loading config")
	}
}
