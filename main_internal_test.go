package main

import (
	"context"
	"flag"
	"os"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	runtimeconf "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type mockDefinition struct {
	Arguments   []interface{}
	ReturnValue interface{}
}

func getNewMockManager(t *testing.T, expectedErrorOnNewManager error) *MockControllerManager {
	k8sManager := NewMockControllerManager(t)
	ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
		return k8sManager, expectedErrorOnNewManager
	}
	ctrl.SetLogger = func(l logr.Logger) {
	}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	return k8sManager
}

func Test_noSpamKey(t *testing.T) {
	t.Run("should not return the same key", func(t *testing.T) {
		// when
		first := noSpamKey(nil)
		second := noSpamKey(nil)

		// then
		assert.NotEqual(t, first, second)
	})
}

func Test_noAggregationKey(t *testing.T) {
	t.Run("should not return the same keys", func(t *testing.T) {
		// when
		firstAggregateKey, firstLocalKey := noAggregationKey(nil)
		secondAggregateKey, secondLocalKey := noAggregationKey(nil)

		// then
		assert.NotEqual(t, firstAggregateKey, secondAggregateKey)
		assert.NotEqual(t, firstLocalKey, secondLocalKey)
	})
}

func Test_startDoguOperator(t *testing.T) {

	t.Run("should fail on missing namespace environment variable", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		defer resetFunc()

		_ = os.Unsetenv("NAMESPACE")

		// when
		err := startDoguOperator()

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read namespace: failed to get env var [NAMESPACE]")
	})

	t.Run("should succeed", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		mockManager.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		mockManager.EXPECT().Start(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		require.NoError(t, err)
	})

	t.Run("should fail on creating dogu reconciler", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, assert.AnError, nil, nil, nil, nil)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail on setting up dogu reconciler", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, assert.AnError, nil, nil)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail on creating global config reconciler", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, assert.AnError, nil, nil, nil)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail on setting up global config reconciler", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, assert.AnError)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail on setting up dogu restart reconciler", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, assert.AnError, nil)
		defer resetFunc()

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)

		// when
		err := startDoguOperator()

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail with error on manager creation", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		defer resetFunc()

		setupEnvironment(t)

		_ = getNewMockManager(t, assert.AnError)

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create manager")
	})
	t.Run("fail setup when error on Add", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		t.Cleanup(resetFunc)

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(assert.AnError)

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail setup when error on AddHealthzCheck", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		t.Cleanup(resetFunc)

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(assert.AnError)

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail setup when error on AddReadyzCheck", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		t.Cleanup(resetFunc)

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		mockManager.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(assert.AnError)

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail setup when error on Start", func(t *testing.T) {
		// given
		resetFunc := setupOverrides(t, nil, nil, nil, nil, nil)
		t.Cleanup(resetFunc)

		setupEnvironment(t)

		mockManager := getNewMockManager(t, nil)
		mockManager.EXPECT().GetConfig().Return(&rest.Config{})
		mockManager.EXPECT().GetEventRecorderFor(mock.Anything).Return(record.NewFakeRecorder(10))
		mockManager.EXPECT().GetClient().Return(fake.NewClientBuilder().WithScheme(getTestScheme()).Build())
		mockManager.EXPECT().Add(mock.Anything).Return(nil)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		mockManager.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		mockManager.EXPECT().Start(mock.Anything).Return(assert.AnError)

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}

func setupEnvironment(t *testing.T) {
	t.Setenv("NAMESPACE", "mynamespace")
	t.Setenv("DOGU_REGISTRY_ENDPOINT", "mynamespace")
	t.Setenv("DOGU_REGISTRY_USERNAME", "mynamespace")
	t.Setenv("DOGU_REGISTRY_PASSWORD", "mynamespace")
	t.Setenv("NETWORK_POLICIES_ENABLED", "true")
}

func setupOverrides(t *testing.T, doguRecErr, globalConfigRecErr, doguRecSetupErr, doguRestartRecSetupErr, globalConfigRecSetupErr error) func() {
	// override default controller method to create a new manager
	oldNewManagerDelegate := ctrl.NewManager

	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return &rest.Config{}
	}

	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	ctrl.GetConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	// override default controller method to signal the setup handler
	oldHandler := ctrl.SetupSignalHandler
	ctrl.SetupSignalHandler = func() context.Context {
		return context.TODO()
	}

	// override default controller-builder and add skipNameValidation for tests
	oldCtrlBuilder := ctrl.NewControllerManagedBy
	ctrl.NewControllerManagedBy = func(m manager.Manager) *ctrl.Builder {
		builder := oldCtrlBuilder(m)
		skipNameValidation := true
		builder.WithOptions(controller.Options{SkipNameValidation: &skipNameValidation})

		return builder
	}

	// override default controller method to retrieve a kube config
	oldSetLoggerDelegate := ctrl.SetLogger

	oldDoguReconciler := NewDoguReconciler
	NewDoguReconciler = func(client client.Client, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder, doguHealthStatusUpdater health.DoguHealthStatusUpdater, availabilityChecker *health.AvailabilityChecker) (controllers.DoguReconciler, error) {
		reconciler := NewMockDoguReconciler(t)
		if doguRecErr == nil {
			reconciler.EXPECT().SetupWithManager(mock.Anything, mock.Anything).Times(1).Return(doguRecSetupErr)
		}
		return reconciler, doguRecErr
	}

	oldGlobalConfigReconciler := NewGlobalConfigReconciler
	NewGlobalConfigReconciler = func(ecosystemClient doguClient.EcoSystemV2Interface, client client.Client, namespace string, doguEvents chan<- event.TypedGenericEvent[*v2.Dogu]) (controllers.GenericReconciler, error) {
		reconciler := NewMockGenericReconciler(t)
		if globalConfigRecErr == nil {
			reconciler.EXPECT().SetupWithManager(mock.Anything).Times(1).Return(globalConfigRecSetupErr)
		}
		return reconciler, globalConfigRecErr
	}

	oldDoguRestartReconciler := NewDoguRestartReconciler
	NewDoguRestartReconciler = func(doguRestartInterface doguClient.DoguRestartInterface, doguInterface doguClient.DoguInterface, recorder record.EventRecorder, gc controllers.DoguRestartGarbageCollector) controllers.GenericReconciler {
		reconciler := NewMockGenericReconciler(t)
		reconciler.EXPECT().SetupWithManager(mock.Anything).Times(1).Return(doguRestartRecSetupErr)
		return reconciler
	}

	return func() {
		ctrl.NewManager = oldNewManagerDelegate
		ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate
		ctrl.GetConfig = oldGetConfigDelegate
		ctrl.SetupSignalHandler = oldHandler
		ctrl.SetLogger = oldSetLoggerDelegate
		ctrl.NewControllerManagedBy = oldCtrlBuilder
		NewDoguReconciler = oldDoguReconciler
		NewDoguRestartReconciler = oldDoguRestartReconciler
		NewGlobalConfigReconciler = oldGlobalConfigReconciler
	}
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogu",
	}, &v2.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogurestart",
	}, &v2.DoguRestart{})
	return scheme
}

func createMockDefinitions() map[string]mockDefinition {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogu",
	}, &v2.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogurestart",
	}, &v2.DoguRestart{})
	myClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	return map[string]mockDefinition{
		"GetScheme":            {ReturnValue: scheme},
		"GetClient":            {ReturnValue: myClient},
		"GetCache":             {ReturnValue: nil},
		"GetConfig":            {ReturnValue: &rest.Config{}},
		"Add":                  {Arguments: []interface{}{mock.Anything}, ReturnValue: nil},
		"AddHealthzCheck":      {Arguments: []interface{}{mock.Anything, mock.Anything}, ReturnValue: nil},
		"AddReadyzCheck":       {Arguments: []interface{}{mock.Anything, mock.Anything}, ReturnValue: nil},
		"Start":                {Arguments: []interface{}{mock.Anything}, ReturnValue: nil},
		"GetControllerOptions": {ReturnValue: runtimeconf.Controller{}},
		"GetEventRecorderFor":  {Arguments: []interface{}{mock.Anything}, ReturnValue: nil},
	}
}
