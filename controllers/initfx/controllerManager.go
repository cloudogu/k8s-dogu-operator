package initfx

import (
	"context"
	"flag"
	"fmt"
	"os"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.uber.org/fx"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
	// set up the logger before the actual logger is instantiated
	// the logger will be replaced later-on with a more sophisticated instance
	startupLog = ctrl.Log.WithName("k8s-dogu-operator")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(doguv2.AddToScheme(scheme))
}

func NewControllerManager(
	lc fx.Lifecycle,
	logger logr.Logger,
	options manager.Options,
	restConfig *rest.Config,
) (manager.Manager, error) {
	ctrl.SetLogger(logger)

	k8sManager, err := ctrl.NewManager(restConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	err = addChecks(k8sManager)
	if err != nil {
		return nil, fmt.Errorf("failed to add checks to the manager: %w", err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	lc.Append(fx.StartHook(func() {
		go func() {
			startupLog.Info("starting manager")
			err := k8sManager.Start(ctx)
			if err != nil {
				startupLog.Error(err, "failed to start manager")
			}
		}()
	}))
	lc.Append(fx.StopHook(func() {
		cancelFunc()
	}))

	return k8sManager, nil
}

type Args []string

var GetArgs = getArgs

func getArgs() Args {
	return os.Args
}

var NewOperatorConfig = config.NewOperatorConfig

func NewManagerOptions(args Args, operatorConfig *config.OperatorConfig) (manager.Options, error) {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	metricsAddr := flags.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	probeAddr := flags.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	enableLeaderElection := flags.Bool("leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	err := flags.Parse(args[1:])
	if err != nil {
		return manager.Options{}, fmt.Errorf("failed to parse command line flags: %w", err)
	}

	return ctrl.Options{
		Scheme:  scheme,
		Metrics: server.Options{BindAddress: *metricsAddr},
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
			operatorConfig.Namespace: {},
		}},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: *probeAddr,
		LeaderElection:         *enableLeaderElection,
		LeaderElectionID:       "951e217a.cloudogu.com",
		EventBroadcaster: record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{
			// This fixes the problem that different events with the same reason get aggregated.
			// Now only events that are exactly the same get aggregated.
			KeyFunc: noAggregationKey,
			// This will prevent any events of the dogu operator from being dropped by the spam filter.
			SpamKeyFunc: noSpamKey,
		}),
	}, nil
}

func noAggregationKey(_ *v1.Event) (string, string) {
	uniqueEventGroup := uuid.NewString()
	return uniqueEventGroup, uniqueEventGroup
}

func noSpamKey(_ *v1.Event) string {
	return uuid.NewString()
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
