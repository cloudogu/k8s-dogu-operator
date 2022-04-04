/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	cesregistry "github.com/cloudogu/cesapp/v4/registry"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	registryUsername     = os.Getenv("CES_REGISTRY_USER")
	registryPassword     = os.Getenv("CES_REGISTRY_PASS")
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
	// namespaceEnvVar is the constant for env variable NAMESPACE
	// which specifies the Namespace to the dogu operator operates in.
	// An empty value means the operator is running with cluster scope.
	namespaceEnvVar = "NAMESPACE"
	// logModeEnvVar is the constant for env variable ZAP_DEVELOPMENT_MODE
	// which specifies the development mode for zap options. Valid values are
	// true or false. In development mode the logger produces stacktraces on warnings and no smapling.
	// In regular mode (default) the logger produces stacktraces on errors and sampling
	logModeEnvVar = "ZAP_DEVELOPMENT_MODE"
)

// applicationExiter is responsible for exiting the application correctly.
type applicationExiter interface {
	// Exit exits the application and prints the actuator error to the console.
	Exit(err error)
}

type osExiter struct {
}

// Exit prints the actual error to stout and exits the application properly.
func (e *osExiter) Exit(err error) {
	setupLog.Error(err, "exiting dogu operator because of error")
	os.Exit(1)
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	exiter := &osExiter{}
	options := getK8sManagerOptions(exiter)

	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		exiter.Exit(err)
	}

	configureManager(k8sManager, exiter, options)

	startK8sManager(k8sManager, exiter)
}

func configureManager(k8sManager manager.Manager, exister applicationExiter, options manager.Options) {
	configureReconciler(k8sManager, exister, options)

	//+kubebuilder:scaffold:builder
	addChecks(k8sManager, exister)
}

func getK8sManagerOptions(exiter applicationExiter) manager.Options {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	configureLogger(exiter)

	watchNamespace, err := getEnvVar(namespaceEnvVar)
	if err != nil {
		exiter.Exit(err)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		Namespace:              watchNamespace,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "951e217a.cloudogu.com",
	}

	return options
}

func configureLogger(exiter applicationExiter) {
	logMode := false
	logModeEnv, err := getEnvVar(logModeEnvVar)

	if err == nil {
		logMode, err = strconv.ParseBool(logModeEnv)
		if err != nil {
			exiter.Exit(fmt.Errorf("failed to parse %s; valid values are true or false: %w", logModeEnv, err))
		}
	}

	opts := zap.Options{
		Development: logMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func startK8sManager(k8sManager manager.Manager, exiter applicationExiter) {
	setupLog.Info("starting manager")
	if err := k8sManager.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		exiter.Exit(err)
	}
}

func configureReconciler(k8sManager manager.Manager, exiter applicationExiter, options manager.Options) {
	doguManager := createDoguManager(k8sManager, exiter, options)
	if err := (controllers.NewDoguReconciler(k8sManager.GetClient(), k8sManager.GetScheme(), doguManager)).SetupWithManager(k8sManager); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Dogu")
		exiter.Exit(err)
	}
}

func createDoguManager(k8sManager manager.Manager, exiter applicationExiter, options manager.Options) *controllers.DoguManager {
	doguRegistry := controllers.NewHTTPDoguRegistry(registryUsername, registryPassword, "https://dogu.cloudogu.com/api/v2/dogus")
	imageRegistry := controllers.NewCraneContainerImageRegistry(registryUsername, registryPassword)
	resourceGenerator := controllers.NewResourceGenerator(k8sManager.GetScheme())
	registry, err := cesregistry.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", options.Namespace)},
	})

	if err != nil {
		setupLog.Error(err, "unable to create registry")
		exiter.Exit(err)
	}

	doguRegistrator := controllers.NewCESDoguRegistrator(k8sManager.GetClient(), registry, resourceGenerator)
	return controllers.NewDoguManager(k8sManager.GetClient(), k8sManager.GetScheme(), resourceGenerator, doguRegistry, imageRegistry, doguRegistrator)
}

func addChecks(mgr manager.Manager, exiter applicationExiter) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		exiter.Exit(err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		exiter.Exit(err)
	}
}

// getEnvVar returns the namespace the operator should be watching for changes
func getEnvVar(name string) (string, error) {
	ns, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("%s must be set", name)
	}
	return ns, nil
}
