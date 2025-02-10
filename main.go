package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"

	"github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/garbagecollection"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/logging"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/google/uuid"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	// +kubebuilder:scaffold:imports

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
	// set up the logger before the actual logger is instantiated
	// the logger will be replaced later-on with a more sophisticated instance
	startupLog           = ctrl.Log.WithName("k8s-dogu-operator")
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
	utilruntime.Must(k8sv2.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	err := startDoguOperator()
	if err != nil {
		startupLog.Error(err, "failed to operate dogu operator")
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
	correlatorOptions := record.CorrelatorOptions{
		// This fixes the problem that different events with the same reason get aggregated.
		// Now only events that are exactly the same get aggregated.
		KeyFunc: noAggregationKey,
		// This will prevent any events of the dogu operator from being dropped by the spam filter.
		SpamKeyFunc: noSpamKey,
	}
	options.EventBroadcaster = record.NewBroadcasterWithCorrelatorOptions(correlatorOptions)

	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	mgrSet, err := configureManager(k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("failed to configure manager: %w", err)
	}

	// print starting info to stderr; we don't use the logger here because by default the level must be ERROR
	println("Starting manager...")

	err = startK8sManager(k8sManager)

	// TODO: Start here
	ctx := context.Background()

	dogus, err := getInstalledDogus(ctx, k8sManager.GetClient())
	if err != nil {
		fmt.Printf("============ERRRR")
		fmt.Printf(err.Error())
		panic(err.Error())
	} else {

		for _, d := range dogus.Items {
			coreDogu, err := mgrSet.LocalDoguFetcher.FetchInstalled(ctx, d.GetSimpleDoguName())
			if err != nil {
				fmt.Printf("==========ERRRR")
				fmt.Printf(err.Error())
				continue
			}
			doguResource := &k8sv2.Dogu{}
			err = mgrSet.Client.Get(ctx, types.NamespacedName{
				Namespace: "ecosystem",
				Name:      coreDogu.GetSimpleName(),
			}, doguResource)
			if err != nil {
				fmt.Printf("==========ERRRR")
				fmt.Printf(err.Error())
				continue
			}

			_, err = mgrSet.ResourceUpserter.UpsertDoguDeployment(
				ctx,
				doguResource,
				coreDogu,
				func(deployment *appsv1.Deployment) {
				},
			)
			if err != nil {
				fmt.Printf("=========================ERROR")
				fmt.Printf(err.Error())
			}

		}

	}

	return err
}

func noAggregationKey(_ *v1.Event) (string, string) {
	uniqueEventGroup := uuid.NewString()
	return uniqueEventGroup, uniqueEventGroup
}

func noSpamKey(_ *v1.Event) string {
	return uuid.NewString()
}

func configureManager(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) (*util.ManagerSet, error) {
	ecosystemClientSet, err := getEcoSystemClientSet(k8sManager.GetConfig())
	if err != nil {
		return nil, err
	}

	k8sClientSet, err := getK8sClientSet(k8sManager.GetConfig())
	if err != nil {
		return nil, err
	}

	availabilityChecker := &health.AvailabilityChecker{}
	eventRecorder := k8sManager.GetEventRecorderFor("k8s-dogu-operator")
	healthStatusUpdater := health.NewDoguStatusUpdater(ecosystemClientSet, eventRecorder, k8sClientSet)

	if err = resourceRequirementsUpdater(k8sManager, operatorConfig.Namespace, k8sClientSet); err != nil {
		return nil, fmt.Errorf("failed to create resource requirements updater: %w", err)
	}

	mgrSet, err := configureReconciler(k8sManager, k8sClientSet, ecosystemClientSet, healthStatusUpdater, availabilityChecker, operatorConfig, eventRecorder)
	if err != nil {
		return nil, fmt.Errorf("failed to configure reconciler: %w", err)
	}

	err = addRunners(k8sManager, k8sClientSet, ecosystemClientSet, healthStatusUpdater, availabilityChecker, operatorConfig.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to add runners: %w", err)
	}

	// +kubebuilder:scaffold:builder
	err = addChecks(k8sManager)
	if err != nil {
		return nil, fmt.Errorf("failed to add checks to the manager: %w", err)
	}

	return mgrSet, nil
}

func getK8sManagerOptions(operatorConfig *config.OperatorConfig) manager.Options {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	options := ctrl.Options{
		Scheme:  scheme,
		Metrics: server.Options{BindAddress: metricsAddr},
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
			operatorConfig.Namespace: {},
		}},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "951e217a.cloudogu.com",
	}

	return options
}

func getInstalledDogus(ctx context.Context, cl k8sclient.Client) (*k8sv2.DoguList, error) {
	doguList := &k8sv2.DoguList{}

	err := cl.List(ctx, doguList, k8sclient.InNamespace("ecosystem"))
	if err != nil {
		return nil, fmt.Errorf("failed to list dogus in namespace [%s]: %w", "ecosystem", err)
	}

	return doguList, nil
}

func startK8sManager(k8sManager manager.Manager) error {
	startupLog.Info("starting manager")
	err := k8sManager.Start(ctrl.SetupSignalHandler())
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	return nil
}

func resourceRequirementsUpdater(k8sManager manager.Manager, namespace string, clientSet kubernetes.Interface) error {
	configMapClient := clientSet.CoreV1().ConfigMaps(namespace)
	localDoguFetcher := cesregistry.NewLocalDoguFetcher(dogu.NewDoguVersionRegistry(configMapClient), dogu.NewLocalDoguDescriptorRepository(configMapClient))

	requirementsUpdater, err := resource.NewRequirementsUpdater(k8sManager.GetClient(), namespace, repository.NewDoguConfigRepository(configMapClient), localDoguFetcher, repository.NewGlobalConfigRepository(configMapClient))
	if err != nil {
		return err
	}

	if err := k8sManager.Add(requirementsUpdater); err != nil {
		return fmt.Errorf("failed to add requirementsUpdater as runnable to the manager: %w", err)
	}

	return nil
}

func configureReconciler(k8sManager manager.Manager, k8sClientSet controllers.ClientSet,
	ecosystemClientSet *ecoSystem.EcoSystemV2Client, healthStatusUpdater health.DoguHealthStatusUpdater,
	availabilityChecker *health.AvailabilityChecker, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder) (*util.ManagerSet, error) {

	localDoguFetcher := cesregistry.NewLocalDoguFetcher(
		dogu.NewDoguVersionRegistry(k8sClientSet.CoreV1().ConfigMaps(operatorConfig.Namespace)),
		dogu.NewLocalDoguDescriptorRepository(k8sClientSet.CoreV1().ConfigMaps(operatorConfig.Namespace)),
	)

	doguManager, mgrSet, err := controllers.NewManager(
		k8sManager.GetClient(),
		ecosystemClientSet,
		operatorConfig,
		eventRecorder,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dogu manager: %w", err)
	}

	doguReconciler, err := controllers.NewDoguReconciler(
		k8sManager.GetClient(),
		ecosystemClientSet.Dogus(operatorConfig.Namespace),
		doguManager,
		eventRecorder,
		operatorConfig.Namespace,
		localDoguFetcher,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new dogu reconciler: %w", err)
	}

	err = doguReconciler.SetupWithManager(k8sManager)
	if err != nil {
		return nil, fmt.Errorf("failed to setup dogu reconciler with manager: %w", err)
	}

	deploymentReconciler := controllers.NewDeploymentReconciler(
		k8sClientSet,
		availabilityChecker,
		healthStatusUpdater,
		localDoguFetcher,
	)
	err = deploymentReconciler.SetupWithManager(k8sManager)
	if err != nil {
		return nil, fmt.Errorf("failed to setup deployment reconciler with manager: %w", err)
	}

	restartInterface := ecosystemClientSet.DoguRestarts(operatorConfig.Namespace)
	if err = controllers.NewDoguRestartReconciler(restartInterface, ecosystemClientSet.Dogus(operatorConfig.Namespace), eventRecorder, garbagecollection.NewDoguRestartGarbageCollector(restartInterface)).
		SetupWithManager(k8sManager); err != nil {
		return nil, fmt.Errorf("failed to setup dogu restart reconciler with manager: %w", err)
	}

	// +kubebuilder:scaffold:builder

	return mgrSet, nil
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

func addRunners(k8sManager manager.Manager, k8sClientSet controllers.ClientSet,
	ecosystemClientSet ecoSystem.EcoSystemV2Interface, updater health.DoguHealthStatusUpdater,
	availabilityChecker *health.AvailabilityChecker, namespace string) error {
	doguInterface := ecosystemClientSet.Dogus(namespace)
	deploymentInterface := k8sClientSet.AppsV1().Deployments(namespace)
	healthStartupHandler := health.NewStartupHandler(doguInterface, deploymentInterface, availabilityChecker, updater)
	err := k8sManager.Add(healthStartupHandler)
	if err != nil {
		return err
	}

	healthShutdownHandler := health.NewShutdownHandler(doguInterface)
	err = k8sManager.Add(healthShutdownHandler)
	if err != nil {
		return err
	}

	return nil
}

func getEcoSystemClientSet(config *rest.Config) (*ecoSystem.EcoSystemV2Client, error) {
	ecosystemClientSet, err := ecoSystem.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ecosystem client set: %w", err)
	}

	return ecosystemClientSet, nil
}

func getK8sClientSet(config *rest.Config) (controllers.ClientSet, error) {
	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client set: %w", err)
	}

	return k8sClientSet, nil
}
