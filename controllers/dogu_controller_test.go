package controllers

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ctx = context.TODO()

func Test_evaluateRequiredOperation(t *testing.T) {
	t.Run("installed should return upgrade", func(t *testing.T) {
		// given
		testDoguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv1.DoguSpec{Name: "official/ledogu", Version: "9000.0.0-1"},
			Status: k8sv1.DoguStatus{
				Status: k8sv1.DoguStatusInstalled,
			},
		}

		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled}
		recorder := mocks.NewEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := new(mocks.LocalDoguFetcher)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             nil,
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(ctx, testDoguCr)

		// then
		require.NoError(t, err)
		localDoguFetcher.AssertExpectations(t)
		recorder.AssertExpectations(t)
		assert.Equal(t, Upgrade, operation)
	})
	t.Run("installed should return ignore for any other changes on a pre-existing dogu resource", func(t *testing.T) {
		// given
		testDoguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv1.DoguSpec{Name: "official/ledogu", Version: "42.0.0-1"},
			Status: k8sv1.DoguStatus{
				Status: k8sv1.DoguStatusInstalled,
			},
		}

		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled}
		recorder := mocks.NewEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := new(mocks.LocalDoguFetcher)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             nil,
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(ctx, testDoguCr)

		// then
		require.NoError(t, err)
		localDoguFetcher.AssertExpectations(t)
		recorder.AssertExpectations(t)
		assert.Equal(t, Ignore, operation)
	})
	t.Run("installed should fail because of version parsing errors", func(t *testing.T) {
		// given
		testDoguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv1.DoguSpec{Name: "official/ledogu", Version: "lol.I.don't.care-äöüß"},
			Status: k8sv1.DoguStatus{
				Status: k8sv1.DoguStatusInstalled,
			},
		}

		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled}
		recorder := mocks.NewEventRecorder(t)
		recorder.On("Eventf", testDoguCr, v1.EventTypeWarning, operatorEventReason, mock.Anything, mock.Anything)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := new(mocks.LocalDoguFetcher)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             nil,
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(ctx, testDoguCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse major version")
		localDoguFetcher.AssertExpectations(t)
		recorder.AssertExpectations(t)
		assert.Equal(t, Ignore, operation)
	})

	// TODO: Joshua will be so kind and will clean up after me with his deliciously refactored tests
	// t.Run("deletiontimestamp should return delete", func(t *testing.T) {
	// 	now := v1.NewTime(time.Now())
	// 	testDoguCr.DeletionTimestamp = &now
	//
	// 	operation, err := evaluateRequiredOperation(nil, testDoguCr)
	//
	// 	require.NoError(t, err)
	// 	assert.Equal(t, Delete, operation)
	// 	testDoguCr.DeletionTimestamp = nil
	// })
	//
	// t.Run("installing should return ignore", func(t *testing.T) {
	// 	testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalling}
	//
	// 	operation, err := evaluateRequiredOperation(nil, testDoguCr)
	//
	// 	require.NoError(t, err)
	// 	assert.Equal(t, Ignore, operation)
	// })
	//
	// t.Run("deleting should return ignore", func(t *testing.T) {
	// 	testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting}
	//
	// 	operation, err := evaluateRequiredOperation(nil, testDoguCr)
	//
	// 	require.NoError(t, err)
	// 	assert.Equal(t, Ignore, operation)
	// })
	//
	// t.Run("default should return ignore", func(t *testing.T) {
	// 	testDoguCr.Status = k8sv1.DoguStatus{Status: "youaresomethingelse"}
	//
	// 	operation, err := evaluateRequiredOperation(nil, testDoguCr)
	//
	// 	require.NoError(t, err)
	// 	assert.Equal(t, Ignore, operation)
	// })
}
