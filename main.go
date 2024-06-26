package main

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

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
	// Configuration holds all the operator-wide configurable settings.
	Configuration struct {
		LeaderElection    bool   `koanf:"enable-leader-election"`
		MetricsAddr       string `koanf:"metrics-addr"`
		ReconcileInterval string `koanf:"reconcile-interval"`
		Debug             bool   `koanf:"verbose"`
	}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(syncv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	loadConfig()
	setupLogger()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: config.MetricsAddr,
		},
		LeaderElection:   config.LeaderElection,
		LeaderElectionID: "bd39f6a0.appuio.ch",
		// Limit the manager to only watch the given namespace
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = map[string]cache.Config{
				getWatchNamespace(): {},
			}
			return cache.New(config, opts)
		},
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

	setupLog.V(1).Info("Configuration from flags", "config", config)

	supplier := func() *controllers.SyncConfigReconciler {
		return &controllers.SyncConfigReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Namespace"),
			Scheme: mgr.GetScheme(),
		}
	}
	mainScr := supplier()
	mainScr.ReconcileInterval = interval
	mainScr.WatchNamespace = getWatchNamespace()
	mainScr.Log = ctrl.Log.WithName("controllers").WithName("SyncConfig")

	if err = mainScr.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SyncConfig")
		os.Exit(1)
	}

	if err = (&controllers.NamespaceReconciler{
		Client:                  mgr.GetClient(),
		Log:                     ctrl.Log.WithName("controllers").WithName("Namespace"),
		Scheme:                  mgr.GetScheme(),
		NewSyncConfigReconciler: supplier,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
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
func loadConfig() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.String("metrics-addr", config.MetricsAddr, "The address the metric endpoint binds to.")
	f.Bool("enable-leader-election", config.LeaderElection, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	f.String("reconcile-interval", config.ReconcileInterval, "The interval of which SyncConfigs get reconciled.")
	f.BoolP("verbose", "v", config.Debug, "Enable debug mode")
	f.Usage = func() {
		fmt.Println("Usage of Espejo:")
		fmt.Print(f.FlagUsages())
		os.Exit(0)
	}
	if err := f.Parse(os.Args[1:]); err != nil {
		setupLog.Error(err, "Could not parse flags.")
		os.Exit(1)
	}
	if err := koanfInstance.Load(posflag.Provider(f, ".", koanfInstance), nil); err != nil {
		setupLog.Error(err, "Could not configure settings from flags.")
		os.Exit(1)
	}
	if err := koanfInstance.Unmarshal("", &config); err != nil {
		setupLog.Error(err, "Could not unmarshal config.")
		os.Exit(1)
	}
}

func setupLogger() {
	logLevel := zapcore.InfoLevel
	if config.Debug {
		logLevel = zapcore.DebugLevel
	}
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(logLevel)))
}
