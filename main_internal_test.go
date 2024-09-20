package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type mockDefinition struct {
	Arguments   []interface{}
	ReturnValue interface{}
}

func getCopyMap(definitions map[string]mockDefinition) map[string]mockDefinition {
	newCopyMap := map[string]mockDefinition{}
	for k, v := range definitions {
		newCopyMap[k] = v
	}
	return newCopyMap
}

func getNewMockManager(expectedErrorOnNewManager error, definitions map[string]mockDefinition) manager.Manager {
	k8sManager := &extMocks.ControllerManager{}
	ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
		for key, value := range definitions {
			k8sManager.On(key, value.Arguments...).Return(value.ReturnValue)
		}
		return k8sManager, expectedErrorOnNewManager
	}
	ctrl.SetLogger = func(l logr.Logger) {
		k8sManager.Mock.On("GetLogger").Return(l)
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

	t.Run("Error on missing namespace environment variable", func(t *testing.T) {
		// given
		resetFunc := setupOverrides()
		defer resetFunc()

		_ = os.Unsetenv("NAMESPACE")
		getNewMockManager(nil, createMockDefinitions())

		// when
		err := startDoguOperator()

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read namespace: failed to get env var [NAMESPACE]")
	})

	t.Run("Test without logger environment variables", func(t *testing.T) {
		// given
		resetFunc := setupOverrides()
		defer resetFunc()

		setupEnvironment(t)

		k8sManager := getNewMockManager(nil, createMockDefinitions())

		// when
		err := startDoguOperator()

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, k8sManager)
	})

	expectedError := fmt.Errorf("this is my expected error")

	t.Run("Test with error on manager creation", func(t *testing.T) {
		// given
		resetFunc := setupOverrides()
		defer resetFunc()

		setupEnvironment(t)

		getNewMockManager(expectedError, createMockDefinitions())

		// when
		err := startDoguOperator()

		// then
		require.ErrorIs(t, err, expectedError)
	})

	mockDefinitionsThatCanFail := []string{
		"Add",
		"AddHealthzCheck",
		"AddReadyzCheck",
		"Start",
	}

	for _, mockDefinitionName := range mockDefinitionsThatCanFail {
		t.Run(fmt.Sprintf("fail setup when error on %s", mockDefinitionName), func(t *testing.T) {
			// given
			resetFunc := setupOverrides()
			t.Cleanup(resetFunc)

			setupEnvironment(t)

			adaptedMockDefinitions := getCopyMap(createMockDefinitions())
			adaptedMockDefinitions[mockDefinitionName] = mockDefinition{
				Arguments:   adaptedMockDefinitions[mockDefinitionName].Arguments,
				ReturnValue: expectedError,
			}
			getNewMockManager(nil, adaptedMockDefinitions)

			// when
			startErr := startDoguOperator()

			// then
			require.ErrorIs(t, startErr, expectedError)
		})
	}
}

func setupEnvironment(t *testing.T) {
	dockerRegistry := config.DockerRegistrySecretData{
		Auths: map[string]config.DockerRegistryData{
			"my.registry": {
				Username: "myusername",
				Password: "mypassword",
				Email:    "myemail",
				Auth:     "myauth",
			},
		},
	}
	dockerRegistryString, err := json.Marshal(dockerRegistry)
	require.NoError(t, err)
	t.Setenv("NAMESPACE", "mynamespace")
	t.Setenv("DOGU_REGISTRY_ENDPOINT", "mynamespace")
	t.Setenv("DOGU_REGISTRY_USERNAME", "mynamespace")
	t.Setenv("DOGU_REGISTRY_PASSWORD", "mynamespace")
	t.Setenv("DOCKER_REGISTRY", string(dockerRegistryString))
}

func setupOverrides() func() {
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

	oldDoguManager := controllers.NewManager
	controllers.NewManager = func(client client.Client, ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface, operatorConfig *config.OperatorConfig, recorder record.EventRecorder) (*controllers.DoguManager, error) {
		return &controllers.DoguManager{}, nil
	}

	return func() {
		ctrl.NewManager = oldNewManagerDelegate
		ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate
		ctrl.GetConfig = oldGetConfigDelegate
		ctrl.SetupSignalHandler = oldHandler
		ctrl.SetLogger = oldSetLoggerDelegate
		ctrl.NewControllerManagedBy = oldCtrlBuilder
		controllers.NewManager = oldDoguManager
	}
}

func createMockDefinitions() map[string]mockDefinition {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &v1.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "dogurestart",
	}, &v1.DoguRestart{})
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
