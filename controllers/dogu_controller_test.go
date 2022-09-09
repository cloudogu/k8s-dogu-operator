package controllers

import (
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguReconciler(t *testing.T) {
	scheme := getTestScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	doguManagerMock := &mocks.Manager{}
	eventRecorderMock := &mocks.EventRecorder{}

	t.Run("fail when creating reconciler", func(t *testing.T) {
		// given
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
		ctrl.GetConfig = func() (*rest.Config, error) {
			return &rest.Config{}, assert.AnError
		}

		// when
		_, err := NewDoguReconciler(fakeClient, scheme, doguManagerMock, eventRecorderMock, "test")

		// then
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("create without errors", func(t *testing.T) {
		// given
		// override default controller method to retrieve a kube config
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
		ctrl.GetConfig = func() (*rest.Config, error) {
			return &rest.Config{}, nil
		}

		// when
		reconcilder, err := NewDoguReconciler(fakeClient, scheme, doguManagerMock, eventRecorderMock, "test")

		// then
		require.NoError(t, err)
		assert.NotNil(t, reconcilder)
		assert.Equal(t, fakeClient, reconcilder.client)
		assert.Equal(t, scheme, reconcilder.scheme)
		assert.Equal(t, doguManagerMock, reconcilder.doguManager)
		assert.Equal(t, eventRecorderMock, reconcilder.recorder)
		mock.AssertExpectationsForObjects(t, doguManagerMock, eventRecorderMock)
	})
}

func Test_evaluateRequiredOperation(t *testing.T) {
	// todo implement tests
	//testDoguCr := &k8sv1.Dogu{}
	//logger := log.FromContext(context.TODO())
	//
	//t.Run("installed should return upgrade", func(t *testing.T) {
	//	testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled}
	//
	//	operation, err := evaluateRequiredOperation(nil, testDoguCr, logger)
	//
	//	require.NoError(t, err)
	//	assert.Equal(t, Upgrade, operation)
	//})
	//
	//t.Run("deletiontimestamp should return delete", func(t *testing.T) {
	//	now := v1.NewTime(time.Now())
	//	testDoguCr.DeletionTimestamp = &now
	//
	//	operation, err := evaluateRequiredOperation(nil, testDoguCr, logger)
	//
	//	require.NoError(t, err)
	//	assert.Equal(t, Delete, operation)
	//	testDoguCr.DeletionTimestamp = nil
	//})
	//
	//t.Run("installing should return ignore", func(t *testing.T) {
	//	testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalling}
	//
	//	operation, err := evaluateRequiredOperation(nil, testDoguCr, logger)
	//
	//	require.NoError(t, err)
	//	assert.Equal(t, Ignore, operation)
	//})
	//
	//t.Run("deleting should return ignore", func(t *testing.T) {
	//	testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting}
	//
	//	operation, err := evaluateRequiredOperation(nil, testDoguCr, logger)
	//
	//	require.NoError(t, err)
	//	assert.Equal(t, Ignore, operation)
	//})
	//
	//t.Run("default should return ignore", func(t *testing.T) {
	//	testDoguCr.Status = k8sv1.DoguStatus{Status: "youaresomethingelse"}
	//
	//	operation, err := evaluateRequiredOperation(nil, testDoguCr, logger)
	//
	//	require.NoError(t, err)
	//	assert.Equal(t, Ignore, operation)
	//})
}
