package controllers

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

type myRequeueableError struct{}

func (mre myRequeueableError) Requeue() bool {
	return true
}

func (mre myRequeueableError) Error() string {
	return "my requeueable error"
}

func TestDoguRequeueHandler_Handle(t *testing.T) {
	t.Run("handle nil error", func(t *testing.T) {
		// given
		reporter := &mocks.StatusReporter{}
		fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()
		handler := NewDoguRequeueHandler(fakeClient, reporter)
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName"},
			Status:     k8sv1.DoguStatus{},
		}

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, nil)

		// then
		require.NoError(t, err)
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
	})

	t.Run("handle non reportable error", func(t *testing.T) {
		// given
		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()
		handler := NewDoguRequeueHandler(fakeClient, reporter)
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName"},
			Status:     k8sv1.DoguStatus{},
		}
		myError := fmt.Errorf("this is my error")

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "this is my error")
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
		mock.AssertExpectationsForObjects(t, reporter)
	})

	t.Run("error on reporting error", func(t *testing.T) {
		// given
		myReportError := fmt.Errorf("this is my report error")
		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(myReportError)
		fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()
		handler := NewDoguRequeueHandler(fakeClient, reporter)
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName"},
			Status:     k8sv1.DoguStatus{},
		}
		myError := fmt.Errorf("this is my error")

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to report error: this is my report error")
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
		mock.AssertExpectationsForObjects(t, reporter)
	})

	t.Run("handle with requeueable error", func(t *testing.T) {
		// given
		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "k8s.cloudogu.com",
			Version: "v1",
			Kind:    "Dogu",
		}, &k8sv1.Dogu{})
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName"},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		handler := NewDoguRequeueHandler(fakeClient, reporter)
		myError := myRequeueableError{}

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError)

		// then
		require.NoError(t, err)
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		mock.AssertExpectationsForObjects(t, reporter)
	})
}

func TestNewDoguRequeueHandler(t *testing.T) {
	// given
	reporter := &mocks.StatusReporter{}
	fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()

	// when
	handler := NewDoguRequeueHandler(fakeClient, reporter)

	// then
	assert.NotNil(t, handler)
	assert.Implements(t, (*requeueHandler)(nil), handler)
}
