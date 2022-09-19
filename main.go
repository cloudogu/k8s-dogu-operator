package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudogu/cesapp-lib/core"
	reg "github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/logging"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	// +kubebuilder:scaffold:imports
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
)

var (
	// Version of the application
	Version = "0.0.0"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	err := startDoguOperator()
	if err != nil {
		setupLog.Error(err, "failed to operate dogu operator")
		os.Exit(1)
	}
}

func startDoguOperator() error {
	err := logging.ConfigureLogger()
	if err != nil {
		return err
	}

	operatorConfig, err := config.NewOperatorConfig(Version)
	if err != nil {
		return fmt.Errorf("failed to create new operator configuration: %w", err)
	}

	options := getK8sManagerOptions(operatorConfig)
	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	if err = handleHardwareLimitUpdater(k8sManager, operatorConfig.Namespace); err != nil {
		return fmt.Errorf("failed to create hardware limit updater: %w", err)
	}

	err = configureManager(k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("failed to configure manager: %w", err)
	}

	return startK8sManager(k8sManager)
}

func configureManager(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) error {
	err := configureReconciler(k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("failed to configure reconciler: %w", err)
	}

	// +kubebuilder:scaffold:builder
	err = addChecks(k8sManager)
	if err != nil {
		return fmt.Errorf("failed to add checks to the manager: %w", err)
	}

	return nil
}

func getK8sManagerOptions(operatorConfig *config.OperatorConfig) manager.Options {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		Namespace:              operatorConfig.Namespace,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "951e217a.cloudogu.com",
	}

	return options
}

func startK8sManager(k8sManager manager.Manager) error {
	setupLog.Info("starting manager")
	err := k8sManager.Start(ctrl.SetupSignalHandler())
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	return nil
}

func handleHardwareLimitUpdater(k8sManager manager.Manager, namespace string) error {
	hardwareLimitUpdater, err := limit.NewHardwareLimitUpdater(k8sManager.GetClient(), namespace)
	if err != nil {
		return err
	}

	if err := k8sManager.Add(hardwareLimitUpdater); err != nil {
		return fmt.Errorf("failed to add hardwareLimitUpdater as runnable to the manager: %w", err)
	}

	return nil
}

func configureReconciler(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) error {
	cesReg, err := reg.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", operatorConfig.Namespace)},
	})
	if err != nil {
		return fmt.Errorf("failed to create CES registry: %w", err)
	}

	eventRecorder := k8sManager.GetEventRecorderFor("k8s-dogu-operator")

	doguManager, err := controllers.NewManager(k8sManager.GetClient(), operatorConfig, cesReg, eventRecorder)
	if err != nil {
		return fmt.Errorf("failed to create dogu manager: %w", err)
	}

	reconciler, err := controllers.NewDoguReconciler(k8sManager.GetClient(), doguManager, eventRecorder, operatorConfig.Namespace, cesReg.DoguRegistry())
	if err != nil {
		return fmt.Errorf("failed to create new dogu reconciler: %w", err)
	}

	err = reconciler.SetupWithManager(k8sManager)
	if err != nil {
		return fmt.Errorf("failed to setup reconciler with manager: %w", err)
	}

	return nil
}

func addChecks(mgr manager.Manager) error {
	err := mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		return fmt.Errorf("failed to add healthz check: %w", err)
	}

	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		return fmt.Errorf("failed to add readyz check: %w", err)
	}

	return nil
}
