package controllers

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var testCtx = context.TODO()

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
		recorder := extMocks.NewEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := mocks.NewLocalDoguFetcher(t)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             fake.NewClientBuilder().Build(),
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
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
		recorder := extMocks.NewEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := new(mocks.LocalDoguFetcher)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             fake.NewClientBuilder().Build(),
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
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
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Eventf", testDoguCr, v1.EventTypeWarning, operatorEventReason, mock.Anything, mock.Anything)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := mocks.NewLocalDoguFetcher(t)
		localDoguFetcher.On("FetchInstalled", "ledogu").Return(localDogu, nil)

		sut := &doguReconciler{
			client:             fake.NewClientBuilder().Build(),
			doguManager:        nil,
			doguRequeueHandler: nil,
			recorder:           recorder,
			fetcher:            localDoguFetcher,
		}

		// when
		operation, err := sut.evaluateRequiredOperation(testCtx, testDoguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse major version")
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

func Test_doguResourceChangeDebugPredicate_Update(t *testing.T) {
	oldDoguResource := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Generation: 123456789},
		Spec:       k8sv1.DoguSpec{Name: "ns/dogu", Version: "1.2.3-4"}}
	newDoguResource := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Generation: 987654321},
		Spec:       k8sv1.DoguSpec{Name: "ns/dogu", Version: "1.2.3-5"}}

	t.Run("should should return false for dogu installation", func(t *testing.T) {
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", newDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: nil,
			ObjectNew: newDoguResource,
		})

		// then
		require.False(t, actual)
	})
	t.Run("should should return false for dogu deletion", func(t *testing.T) {
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", oldDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: nil,
		})

		// then
		require.False(t, actual)
	})
	t.Run("should should return true for dogu upgrade", func(t *testing.T) {
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", newDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: newDoguResource,
		})

		// then
		require.True(t, actual)
	})
	t.Run("should should return false for no dogu change", func(t *testing.T) {
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", oldDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: oldDoguResource,
		})

		// then
		require.False(t, actual)
	})
}

func Test_buildResourceDiff(t *testing.T) {
	oldDoguResource := &k8sv1.Dogu{Spec: k8sv1.DoguSpec{Name: "ns/dogu", Version: "1.2.3-4"}}
	newDoguResource := &k8sv1.Dogu{Spec: k8sv1.DoguSpec{Name: "ns/dogu", Version: "1.2.3-5"}}

	type args struct {
		objOld client.Object
		objNew client.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "create-diff",
			args: args{objOld: nil, objNew: newDoguResource},
			want: "  any(\n+ \t&v1.Dogu{Spec: v1.DoguSpec{Name: \"ns/dogu\", Version: \"1.2.3-5\"}},\n  )\n",
		},
		{
			name: "upgrade-diff",
			args: args{objOld: oldDoguResource, objNew: newDoguResource},
			want: "  &v1.Dogu{\n  \tTypeMeta:   {},\n  \tObjectMeta: {},\n  \tSpec: v1.DoguSpec{\n  \t\tName:        \"ns/dogu\",\n- \t\tVersion:     \"1.2.3-4\",\n+ \t\tVersion:     \"1.2.3-5\",\n  \t\tResources:   {},\n  \t\tSupportMode: false,\n  \t\t... // 2 identical fields\n  \t},\n  \tStatus: {},\n  }\n",
		},
		{
			name: "delete-diff",
			args: args{objOld: oldDoguResource, objNew: nil},
			want: "  any(\n- \t&v1.Dogu{Spec: v1.DoguSpec{Name: \"ns/dogu\", Version: \"1.2.3-4\"}},\n  )\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := buildResourceDiff(tt.args.objOld, tt.args.objNew)
			assert.Equalf(t,
				tt.want,
				result,
				"buildResourceDiff(%v, %v)", tt.args.objOld, tt.args.objNew)
		})
	}
}

func Test_finishOperation(t *testing.T) {
	result, err := finishOperation()

	assert.Empty(t, result)
	assert.Nil(t, err)
}

func Test_requeueOrFinishOperation(t *testing.T) {
	input := ctrl.Result{
		Requeue: true,
	}

	result, err := requeueOrFinishOperation(input)

	assert.Equal(t, input, result)
	assert.Nil(t, err)
}

func Test_requeueWithError(t *testing.T) {
	result, err := requeueWithError(assert.AnError)

	assert.Empty(t, result)
	assert.Same(t, assert.AnError, err)
}

func Test_operation_toString(t *testing.T) {
	assert.Equal(t, operation("Install"), Install)
	assert.Equal(t, operation("Upgrade"), Upgrade)
	assert.Equal(t, operation("Delete"), Delete)
	assert.Equal(t, operation("Ignore"), Ignore)
}

func Test_doguReconciler_checkForVolumeExpansion(t *testing.T) {
	t.Run("should return false and nil if no pvc is found", func(t *testing.T) {
		// given
		sut := &doguReconciler{client: fake.NewClientBuilder().Build()}
		doguCr := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return false and nil if pvc is found but dogu has no dataVolumeSize property", func(t *testing.T) {
		// given
		doguCr := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta()}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return error on invalid volume size", func(t *testing.T) {
		// given
		doguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv1.DoguSpec{Resources: k8sv1.DoguResources{DataVolumeSize: "wrong"}}}
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta()}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse resource volume size")
		assert.False(t, expand)
	})

	t.Run("should return true if volume size is higher than actual", func(t *testing.T) {
		// given
		doguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv1.DoguSpec{Resources: k8sv1.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("1Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.ResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.NoError(t, err)
		assert.True(t, expand)
	})

	t.Run("should return false if size is equal", func(t *testing.T) {
		// given
		doguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv1.DoguSpec{Resources: k8sv1.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("2Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.ResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return error if size is smaller than actual", func(t *testing.T) {
		// given
		doguCr := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv1.DoguSpec{Resources: k8sv1.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("3Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.ResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid dogu state for dogu [test] as requested volume size is "+
			"[2Gi] while existing volume is [3Gi], shrinking of volumes is not allowed")
		assert.False(t, expand)
	})

	t.Run("error on pvc found", func(t *testing.T) {
		// given
		sut := &doguReconciler{client: fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()}
		doguCr := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}

		// when
		expand, err := sut.checkForVolumeExpansion(context.TODO(), doguCr)

		// then
		require.Error(t, err)
		assert.False(t, expand)
	})
}
