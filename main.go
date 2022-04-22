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
	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	//+kubebuilder:scaffold:imports
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
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := startDoguOperator()
	if err != nil {

	}
}

func startDoguOperator() error {
	configureLogger()

	operatorConfig, err := config.NewOperatorConfig(Version)
	if err != nil {
		return fmt.Errorf("falied to create new operator configuration: %w", err)
	}

	options := getK8sManagerOptions(operatorConfig)
	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("falied to start manager: %w", err)
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

	//+kubebuilder:scaffold:builder
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

func configureLogger() {
	logrusLog := logrus.New()
	logrusLog.SetFormatter(&logrus.TextFormatter{})
	logrusLog.SetLevel(logrus.DebugLevel)

	ctrl.SetLogger(logrusr.New(logrusLog))
}

func startK8sManager(k8sManager manager.Manager) error {
	setupLog.Info("starting manager")
	err := k8sManager.Start(ctrl.SetupSignalHandler())
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	return nil
}

func configureReconciler(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) error {
	doguManager, err := controllers.NewDoguManager(operatorConfig.Version, k8sManager.GetClient(), operatorConfig)
	if err != nil {
		return fmt.Errorf("failed to create dogu manager: %w", err)
	}

	err = (controllers.NewDoguReconciler(k8sManager.GetClient(), k8sManager.GetScheme(), doguManager)).SetupWithManager(k8sManager)
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
