package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	extMocks "github.com/cloudogu/k8s-dogu-operator/v2/internal/thirdParty/mocks"
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

	t.Run("handle no error at all", func(t *testing.T) {
		// given
		doguResource := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv2.DoguStatus{},
		}
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventRecorder := &extMocks.EventRecorder{}
		doguInterfaceMock := mocks.NewDoguInterface(t)

		handler := doguRequeueHandler{
			doguInterface:  doguInterfaceMock,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		onRequeue := func(doguResource *k8sv2.Dogu) error { doguResource.Labels["test"] = "true"; return nil }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, nil, onRequeue)

		// then
		require.NoError(t, err)

		assert.Equal(t, result.RequeueAfter, time.Duration(0))

		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)
	})

	t.Run("handle non reportable error", func(t *testing.T) {
		// given
		doguResource := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv2.DoguStatus{},
		}
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventRecorder := &extMocks.EventRecorder{}
		doguInterfaceMock := mocks.NewDoguInterface(t)

		handler := doguRequeueHandler{
			doguInterface:  doguInterfaceMock,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		onRequeue := func(doguResource *k8sv2.Dogu) error { doguResource.Labels["test"] = "true"; return nil }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, assert.AnError, onRequeue)

		// then
		require.NoError(t, err, assert.AnError)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)
	})

	t.Run("handle with requeueable error", func(t *testing.T) {
		// given
		doguResource := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Namespace: namespace},
			Status:     k8sv2.DoguStatus{},
		}

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
		eventRecorder := &extMocks.EventRecorder{}
		eventRecorder.On("Eventf", mock.Anything, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", "10s")

		doguInterfaceMock := mocks.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(context.Background(), doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv2.Dogu, modifyStatusFn func(k8sv2.DoguStatus) k8sv2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguResource.Status)
				assert.Equal(t, k8sv2.DoguStatus{Status: "", RequeueTime: 10000000000, RequeuePhase: "", Health: "", Stopped: false}, status)
			})

		handler := doguRequeueHandler{
			doguInterface:  doguInterfaceMock,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}
		myError := myRequeueableError{}

		requeueCalled := false
		onRequeue := func(doguResource *k8sv2.Dogu) error {
			requeueCalled = true
			return nil
		}

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError, onRequeue)

		// then
		require.NoError(t, err)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		assert.True(t, requeueCalled, "Requeue was not called.")
		mock.AssertExpectationsForObjects(t, eventRecorder)

		eventList, err := fakeNonCacheClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)

		assert.Len(t, eventList.Items, 0)
	})

	t.Run("handle with multierror error", func(t *testing.T) {
		// given
		doguResource := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Namespace: namespace},
			Status:     k8sv2.DoguStatus{},
		}

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
		eventRecorder := extMocks.NewEventRecorder(t)
		eventRecorder.EXPECT().Eventf(mock.Anything, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", "10s")

		doguInterfaceMock := mocks.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(context.Background(), doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv2.Dogu, modifyStatusFn func(k8sv2.DoguStatus) k8sv2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguResource.Status)
				assert.Equal(t, k8sv2.DoguStatus{Status: "", RequeueTime: 10000000000, RequeuePhase: "", Health: "", Stopped: false}, status)
			})

		handler := doguRequeueHandler{
			doguInterface:  doguInterfaceMock,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		myError := errors.New("my not requeue-able error")
		myError2 := myRequeueableError{}
		var myMultipleErrors error
		myMultipleErrors = errors.Join(myMultipleErrors, myError, myError2)

		requeueCalled := false
		onRequeue := func(doguResource *k8sv2.Dogu) error {
			requeueCalled = true
			return nil
		}

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myMultipleErrors, onRequeue)

		// then
		require.NoError(t, err)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		assert.True(t, requeueCalled, "Requeue was not called.")

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

	eventRecorder := &extMocks.EventRecorder{}
	doguInterfaceMock := mocks.NewDoguInterface(t)

	// when
	handler, err := NewDoguRequeueHandler(doguInterfaceMock, eventRecorder, "mynamespace")

	// then
	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Implements(t, (*cloudogu.RequeueHandler)(nil), handler)
}
