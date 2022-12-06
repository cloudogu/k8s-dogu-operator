package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks/external"
	"github.com/hashicorp/go-multierror"
	"testing"
	"time"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
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
		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(doguResource).Build()
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventRecorder := &external.EventRecorder{}

		handler := doguRequeueHandler{
			client:         fakeClient,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, nil, onRequeue)

		// then
		require.NoError(t, err)

		assert.Equal(t, result.RequeueAfter, time.Duration(0))

		assert.Nil(t, doguResource.Status.StatusMessages)
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)
	})

	t.Run("handle non reportable error", func(t *testing.T) {
		// given
		scheme := getTestScheme()
		doguResource := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myName", Labels: map[string]string{"test": "false"}},
			Status:     k8sv1.DoguStatus{},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
		fakeNonCacheClient := fake2.NewSimpleClientset()
		eventRecorder := &external.EventRecorder{}

		handler := doguRequeueHandler{
			client:         fakeClient,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, assert.AnError, onRequeue)

		// then
		require.NoError(t, err, assert.AnError)
		assert.Equal(t, result.RequeueAfter, time.Duration(0))
		assert.Nil(t, doguResource.Status.StatusMessages)
		assert.Equal(t, "false", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)
	})

	t.Run("handle with requeueable error", func(t *testing.T) {
		// given
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
		eventRecorder := &external.EventRecorder{}
		eventRecorder.On("Eventf", mock.Anything, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", "10s")

		handler := doguRequeueHandler{
			client:         fakeClient,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}
		myError := myRequeueableError{}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myError, onRequeue)

		// then
		require.NoError(t, err)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		assert.Equal(t, "true", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)

		eventList, err := fakeNonCacheClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)

		assert.Len(t, eventList.Items, 0)
	})

	t.Run("handle with multierror error", func(t *testing.T) {
		// given
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
		eventRecorder := &external.EventRecorder{}
		eventRecorder.On("Eventf", mock.Anything, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", "10s")

		handler := doguRequeueHandler{
			client:         fakeClient,
			nonCacheClient: fakeNonCacheClient,
			namespace:      namespace,
			recorder:       eventRecorder,
		}

		myError := errors.New("my not requeue-able error")
		myError2 := myRequeueableError{}
		myMultipleErrors := new(multierror.Error)
		myMultipleErrors.Errors = []error{myError, myError2}

		onRequeue := func(doguResource *k8sv1.Dogu) { doguResource.Labels["test"] = "true" }

		// when
		result, err := handler.Handle(context.Background(), "my context", doguResource, myMultipleErrors, onRequeue)

		// then
		require.NoError(t, err)
		assert.Equal(t, result.RequeueAfter, time.Second*10)
		assert.Equal(t, "true", doguResource.Labels["test"])
		mock.AssertExpectationsForObjects(t, eventRecorder)

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

	eventRecorder := &external.EventRecorder{}
	fakeClient := fake.NewClientBuilder().WithScheme(&runtime.Scheme{}).Build()

	// when
	handler, err := NewDoguRequeueHandler(fakeClient, eventRecorder, "mynamespace")

	// then
	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Implements(t, (*requeueHandler)(nil), handler)
}
