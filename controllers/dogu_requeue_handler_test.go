package controllers

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
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
	namespace := "namespace"

	t.Run("handle nil error", func(t *testing.T) {
		// given

		reporter := &mocks.StatusReporter{}

		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventHandler := &mocks.EventRecorder{}

		handler := doguRequeueHandler{
			doguStatusReporter: reporter,
			client:             fakeClient,
			nonCacheClient:     fakeNonCacheClient,
			namespace:          namespace,
			recorder:           eventHandler,
		}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, nil, onRequeue)

		// then
		require.NoError(t, err)

		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))

		assert.Nil(t, doguResource.Status.StatusMessages)
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, reporter, eventHandler)
	})

	t.Run("handle non reportable error", func(t *testing.T) {
		// given

		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventHandler := &mocks.EventRecorder{}

		handler := doguRequeueHandler{
			doguStatusReporter: reporter,
			client:             fakeClient,
			nonCacheClient:     fakeNonCacheClient,
			namespace:          namespace,
			recorder:           eventHandler,
		}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, assert.AnError, onRequeue)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, reporter, eventHandler)
	})

	t.Run("error on reporting error", func(t *testing.T) {
		// given
		myReportError := fmt.Errorf("this is my report error")
		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(myReportError)

		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventHandler := &mocks.EventRecorder{}

		handler := doguRequeueHandler{
			doguStatusReporter: reporter,
			client:             fakeClient,
			nonCacheClient:     fakeNonCacheClient,
			namespace:          namespace,
			recorder:           eventHandler,
		}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, assert.AnError, onRequeue)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to report error: this is my report error")
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, reporter, eventHandler)
	})

	t.Run("handle with requeueable error", func(t *testing.T) {
		// given
		reporter := &mocks.StatusReporter{}
		reporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}, Namespace: namespace},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()

		event := &v1.Event{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "myName.1241245124", Namespace: namespace},
			Reason:     RequeueEventReason,
			InvolvedObject: v1.ObjectReference{
				Name: "myName",
			},
			Message: "This should be deleted.",
		}

		fakeNonCacheClient := fake2.NewSimpleClientset(event)
		eventHandler := &mocks.EventRecorder{}
		eventHandler.On("Eventf", mock.Anything, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", "10s")

		handler := doguRequeueHandler{
			doguStatusReporter: reporter,
			client:             fakeClient,
			nonCacheClient:     fakeNonCacheClient,
			namespace:          namespace,
			recorder:           eventHandler,
		}
		myError := myRequeueableError{}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError, onRequeue)

		// then
		require.NoError(t, err)
		assert.False(t, result.Requeue)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		mock.AssertExpectationsForObjects(t, reporter)
		assert.Equal(t, "true", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, reporter, eventHandler)

		eventList, err := fakeNonCacheClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)

		assert.Len(t, eventList.Items, 0)
	})
}

func TestNewDoguRequeueHandler(t *testing.T) {
	// given
	oldGetConfig := ctrl.GetConfig
	defer func() { ctrl.GetConfig = oldGetConfig }()
	ctrl.GetConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	reporter := &mocks.StatusReporter{}
	eventHandler := &mocks.EventRecorder{}
	fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()

	// when
	handler, err := NewDoguRequeueHandler(fakeClient, reporter, eventHandler, "mynamespace")

	// then
	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Implements(t, (*requeueHandler)(nil), handler)
}
