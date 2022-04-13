package main

import (
	"context"
	"errors"
	"flag"
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/mocks"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
)

type mockExiter struct {
	Error error `json:"error"`
}

func (e *mockExiter) Exit(err error) {
	e.Error = err
}

var testErr = errors.New("test")

func Test_getK8sManagerOptions(t *testing.T) {

	t.Run("successfully get k8s manager options", func(t *testing.T) {
		t.Setenv("NAMESPACE", "default")

		getK8sManagerOptions(&config.OperatorConfig{DevelopmentLogMode: false})
	})
}

func Test_configureManager(t *testing.T) {
	t.Run("successfully configure manager", func(t *testing.T) {
		exiter := &mockExiter{}
		k8sManager := &mocks.Manager{}
		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "dogu.cloudogu.com",
			Version: "v1",
			Kind:    "dogu",
		}, &v1.Dogu{})
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sManager.Mock.On("GetScheme").Return(scheme)
		k8sManager.Mock.On("GetClient").Return(client)
		k8sManager.Mock.On("GetControllerOptions").Return(v1alpha1.ControllerConfigurationSpec{})
		k8sManager.Mock.On("AddHealthzCheck", mock.Anything, mock.Anything).Return(nil)
		k8sManager.Mock.On("AddReadyzCheck", mock.Anything, mock.Anything).Return(nil)
		logger := logr.Logger{}
		k8sManager.Mock.On("GetLogger").Return(logger.WithSink(&log.NullLogSink{}))
		k8sManager.Mock.On("SetFields", mock.Anything).Return(nil)
		k8sManager.Mock.On("Add", mock.Anything).Return(nil)

		operatorConfig := &config.OperatorConfig{
			Namespace: "myNamespace",
			DoguRegistry: config.DoguRegistryData{
				Endpoint: "myEndpoint",
				Username: "myUsername",
				Password: "myPassword",
			},
		}

		configureManager(k8sManager, operatorConfig, exiter, manager.Options{})

		assert.Nil(t, exiter.Error)
		mock.AssertExpectationsForObjects(t, k8sManager)
	})
}

func Test_startK8sManager(t *testing.T) {

	oldHandler := ctrl.SetupSignalHandler
	defer func() { ctrl.SetupSignalHandler = oldHandler }()
	ctrl.SetupSignalHandler = func() context.Context {
		return context.TODO()
	}

	t.Run("success", func(t *testing.T) {
		exiter := &mockExiter{}
		k8sManager := &mocks.Manager{}

		k8sManager.Mock.On("Start", mock.Anything).Return(nil)

		startK8sManager(k8sManager, exiter)

		assert.Nil(t, exiter.Error)
	})

	t.Run("failed to start", func(t *testing.T) {
		exiter := &mockExiter{}
		k8sManager := &mocks.Manager{}
		k8sManager.Mock.On("Start", mock.Anything).Return(testErr)

		startK8sManager(k8sManager, exiter)

		assert.Error(t, exiter.Error)
	})
}

func Test_addChecks(t *testing.T) {
	t.Run("fail to add health check", func(t *testing.T) {
		exiter := &mockExiter{}
		k8sManager := &mocks.Manager{}
		k8sManager.Mock.On("AddHealthzCheck", mock.Anything, mock.Anything).Return(testErr)
		k8sManager.Mock.On("AddReadyzCheck", mock.Anything, mock.Anything).Return(nil)

		addChecks(k8sManager, exiter)

		assert.Error(t, exiter.Error)
	})

	t.Run("fail to add ready check", func(t *testing.T) {
		exiter := &mockExiter{}
		k8sManager := &mocks.Manager{}
		k8sManager.Mock.On("AddHealthzCheck", mock.Anything, mock.Anything).Return(nil)
		k8sManager.Mock.On("AddReadyzCheck", mock.Anything, mock.Anything).Return(testErr)

		addChecks(k8sManager, exiter)

		assert.Error(t, exiter.Error)
	})
}

func Test_configureLogger(t *testing.T) {
	t.Run("configure logger with log mode env var", func(t *testing.T) {
		t.Setenv("ZAP_DEVELOPMENT_MODE", "true")
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

		operatorConfig := &config.OperatorConfig{DevelopmentLogMode: true}
		configureLogger(operatorConfig)
	})
}
