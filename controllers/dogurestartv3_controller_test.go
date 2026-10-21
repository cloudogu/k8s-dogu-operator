package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/restart"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const v3RestartName = "test-restart"

var (
	v3RestartRequest     = reconcile.Request{Namespace: testNamespace, Name: v3RestartName}
	v3RestartCreatedAt   = metav1.NewTime(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC))
	v3DoguReadyCondition = metav1.Condition{Type: v3beta1.ConditionReady, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded, ObservedGeneration: 1}
)

func restartV3TestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = doguv2.AddToScheme(scheme)
	_ = v3beta1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newRestartV3Client(objects ...client.Object) client.Client {
	return newRestartV3ClientWithFuncs(interceptor.Funcs{}, objects...)
}

func newRestartV3ClientWithFuncs(funcs interceptor.Funcs, objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(restartV3TestScheme()).
		WithObjects(objects...).
		WithStatusSubresource(&doguv2.DoguRestart{}).
		WithInterceptorFuncs(funcs).
		Build()
}

// failStatusUpdate lets every status update fail.
var failStatusUpdate = interceptor.Funcs{
	SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
		return assert.AnError
	},
}

func doguV3Restart(conditions ...metav1.Condition) *doguv2.DoguRestart {
	return &doguv2.DoguRestart{
		Name: v3RestartName, Namespace: testNamespace, CreationTimestamp: v3RestartCreatedAt, Generation: 1,
		Spec:   doguv2.DoguRestartSpec{DoguName: testDoguName},
		Status: doguv2.DoguRestartStatus{Conditions: conditions},
	}
}

func v3Dogu(conditions ...metav1.Condition) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name: testDoguName, Namespace: testNamespace, Generation: 1,
		Spec:   v3beta1.DoguSpec{Name: testDoguName, DoguApiVersion: v3beta1.DoguApiVersionV3},
		Status: v3beta1.DoguStatus{Conditions: conditions},
	}
}

func condition(conditionType string, status metav1.ConditionStatus, reason string) metav1.Condition {
	return metav1.Condition{Type: conditionType, Status: status, Reason: reason}
}

type restartV3Test struct {
	client    client.Client
	recorder  *record.FakeRecorder
	gc        *MockDoguRestartGarbageCollector
	restarter *mockDoguRestarter
	sut       *DoguRestartV3Reconciler
}

// newRestartV3Test creates the reconciler with a restarter mock, that expects no calls at all
func newRestartV3Test(t *testing.T, objects ...client.Object) restartV3Test {
	return newRestartV3TestWithFuncs(t, interceptor.Funcs{}, objects...)
}

// newRestartV3TestWithFuncs creates the reconciler like newRestartV3Test, but with a client that calls the given funcs, e.g. to inject errors.
func newRestartV3TestWithFuncs(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) restartV3Test {
	c := newRestartV3ClientWithFuncs(funcs, objects...)
	recorder := record.NewFakeRecorder(10)
	gc := NewMockDoguRestartGarbageCollector(t)
	restarter := newMockDoguRestarter(t)
	return restartV3Test{client: c, recorder: recorder, gc: gc, restarter: restarter, sut: newDoguRestartV3Reconciler(c, restarter, gc, recorder)}
}

// atRestartCreation matches the creation timestamp of the restart. The fake client returns timestamps in the local
// time zone, so the instant is compared instead of the time.Time value.
func atRestartCreation() any {
	return mock.MatchedBy(func(restartAt time.Time) bool { return restartAt.Equal(v3RestartCreatedAt.Time) })
}

func (rt restartV3Test) expectApply(superseded bool, err error) *mockDoguRestarter_Apply_Call {
	return rt.restarter.EXPECT().Apply(mock.Anything, testNamespace, testDoguName, atRestartCreation()).Return(superseded, err)
}

func (rt restartV3Test) expectRolledOut(rolledOut bool, message string, err error) *mockDoguRestarter_RolledOut_Call {
	return rt.restarter.EXPECT().RolledOut(mock.Anything, testNamespace, testDoguName, atRestartCreation()).Return(rolledOut, message, err)
}

func (rt restartV3Test) persistedRestart(t *testing.T) *doguv2.DoguRestart {
	t.Helper()
	persisted := &doguv2.DoguRestart{}
	require.NoError(t, rt.client.Get(context.Background(), v3RestartRequest.NamespacedName, persisted))
	return persisted
}

func assertCondition(t *testing.T, restart *doguv2.DoguRestart, conditionType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	actual := meta.FindStatusCondition(restart.Status.Conditions, conditionType)
	require.NotNil(t, actual, "condition %s not found", conditionType)
	assert.Equal(t, status, actual.Status, "status of condition %s", conditionType)
	assert.Equal(t, reason, actual.Reason, "reason of condition %s", conditionType)
}

func assertEvent(t *testing.T, recorder *record.FakeRecorder, expected string) {
	t.Helper()
	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, expected)
	default:
		t.Fatalf("expected event %q but got none", expected)
	}
}

func assertNoEvent(t *testing.T, recorder *record.FakeRecorder) {
	t.Helper()
	select {
	case event := <-recorder.Events:
		t.Fatalf("expected no event but got %q", event)
	default:
	}
}

func TestNewDoguRestartV3Reconciler(t *testing.T) {
	t.Run("should register the controller with the manager if dogu v3 is enabled", func(t *testing.T) {
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: new(true)})
		managerMock.EXPECT().GetScheme().Return(restartV3TestScheme())
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(nil)
		managerMock.EXPECT().GetCache().Return(nil)
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		sut, err := NewDoguRestartV3Reconciler(nil, nil, NewMockDoguRestartGarbageCollector(t), managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should not register the controller with the manager if dogu v3 is disabled", func(t *testing.T) {
		// the manager mock fails the test on any call
		sut, err := NewDoguRestartV3Reconciler(nil, nil, NewMockDoguRestartGarbageCollector(t), newMockCtrlManager(t), &opConfig.OperatorConfig{DoguV3Enabled: false})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})
}

func TestDoguRestartV3Reconciler_Reconcile(t *testing.T) {
	t.Run("should do nothing if the restart does not exist", func(t *testing.T) {
		rt := newRestartV3Test(t)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("should ignore restarts that are handled with a phase", func(t *testing.T) {
		r := doguV3Restart()
		r.Status.Phase = doguv2.RestartStatusPhaseStopping
		// the restarter mock fails the test if the dogu is restarted
		rt := newRestartV3Test(t, r, v3Dogu(v3DoguReadyCondition))

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.Empty(t, rt.persistedRestart(t).Status.Conditions)
	})

	t.Run("should collect garbage for terminal restarts", func(t *testing.T) {
		// e.g. the workload was recreated without the annotation; the restarter mock fails the test if a terminal restart restarts again
		rt := newRestartV3Test(t,
			doguV3Restart(condition(doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded),
				condition(doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)),
			v3Dogu(v3DoguReadyCondition))
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assertNoEvent(t, rt.recorder)
	})

	t.Run("should leave new restarts of missing dogus to the v2 controller but check again", func(t *testing.T) {
		// the dogu may be missing only in the cache, while the v2 controller already sees it as v3 dogu and ignores the restart;
		// the requeue ends once the cache contains the dogu or the v2 controller has set a phase
		rt := newRestartV3Test(t, doguV3Restart())

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{RequeueAfter: requeueWaitTimeout}, result)
		assert.Empty(t, rt.persistedRestart(t).Status.Conditions)
	})

	t.Run("should fail permanently if the dogu of a started restart was deleted", func(t *testing.T) {
		rt := newRestartV3Test(t,
			doguV3Restart(condition(doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle)))
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound)
		assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound)
		assertEvent(t, rt.recorder, "Warning DoguNotFound")
	})

	t.Run("should ignore restarts of v2 dogus", func(t *testing.T) {
		dogu := v3Dogu(v3DoguReadyCondition)
		dogu.Spec.DoguApiVersion = v3beta1.DoguApiVersionV2
		// the restarter mock fails the test if the dogu is restarted
		rt := newRestartV3Test(t, doguV3Restart(), dogu)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.Empty(t, rt.persistedRestart(t).Status.Conditions)
	})

	t.Run("should wait while the dogu is not idle", func(t *testing.T) {
		busy := func(reason string) []metav1.Condition {
			return []metav1.Condition{{Type: v3beta1.ConditionReady, Status: metav1.ConditionFalse, Reason: reason, ObservedGeneration: 2}}
		}
		tests := []struct {
			name       string
			conditions []metav1.Condition
			deleting   bool
		}{
			{name: "without Ready condition"},
			// the spec of the dogu was changed but the change is not reconciled yet
			{name: "with outdated Ready condition", conditions: []metav1.Condition{{Type: v3beta1.ConditionReady, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded, ObservedGeneration: 1}}},
			{name: "installing", conditions: busy(v3beta1.ReasonInstalling)},
			{name: "upgrading", conditions: busy(v3beta1.ReasonUpgrading)},
			{name: "resizing PVC", conditions: busy(v3beta1.ReasonResizingPVC)},
			{name: "changing export mode", conditions: busy(v3beta1.ReasonChangingExportMode)},
			{name: "starting", conditions: busy(v3beta1.ReasonStarting)},
			{name: "stopping", conditions: busy(v3beta1.ReasonStopping)},
			{name: "deleting", conditions: busy(v3beta1.ReasonDeleting)},
			{name: "with deletion timestamp", conditions: []metav1.Condition{{Type: v3beta1.ConditionReady, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded, ObservedGeneration: 2}}, deleting: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dogu := v3Dogu(tt.conditions...)
				dogu.Generation = 2
				if tt.deleting {
					// the fake client only accepts objects with a deletion timestamp if they have a finalizer
					dogu.DeletionTimestamp = new(metav1.Now())
					dogu.Finalizers = []string{"test"}
				}
				// the restarter mock fails the test if the dogu is restarted
				rt := newRestartV3Test(t, doguV3Restart(), dogu)

				result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

				require.NoError(t, err)
				assert.Equal(t, reconcile.Result{RequeueAfter: requeueWaitTimeout}, result)
				persisted := rt.persistedRestart(t)
				assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle)
				assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle)
			})
		}
	})

	t.Run("should restart a dogu whose last operation failed", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(),
			v3Dogu(metav1.Condition{Type: v3beta1.ConditionReady, Status: metav1.ConditionFalse, Reason: v3beta1.ReasonUpgradeFailed, ObservedGeneration: 1}))
		rt.expectApply(false, nil)
		rt.expectRolledOut(false, "workloads not rolled out: Deployment/test", nil)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assertCondition(t, rt.persistedRestart(t), doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
	})

	t.Run("should restart with the creation timestamp and wait for the rollout", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		rt.expectApply(false, nil)
		rt.expectRolledOut(false, "workloads not rolled out: Deployment/test", nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		// workload changes trigger the reconciliation, the requeue only catches a rollout that stops emitting them
		assert.Equal(t, reconcile.Result{RequeueAfter: rolloutRequeueInterval}, result)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
		assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress)
		assert.Equal(t, "workloads not rolled out: Deployment/test", meta.FindStatusCondition(persisted.Status.Conditions, doguv2.DoguRestartConditionReady).Message)
		assertEvent(t, rt.recorder, "Normal Restarted")
	})

	t.Run("should be ready once the workloads are rolled out", func(t *testing.T) {
		rt := newRestartV3Test(t,
			doguV3Restart(condition(doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded),
				condition(doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress)),
			v3Dogu(v3DoguReadyCondition))
		rt.expectApply(false, nil)
		rt.expectRolledOut(true, "", nil)
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
		assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
		assertEvent(t, rt.recorder, "Normal Ready")
	})

	t.Run("should emit the Restarted event only once on repeated reconciliation", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		// the restarter does not change workloads that already carry the restart, so applying it again is safe
		rt.expectApply(false, nil).Times(2)
		rt.expectRolledOut(false, "workloads not rolled out: Deployment/test", nil).Times(2)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)
		require.NoError(t, err)
		_, err = rt.sut.Reconcile(testCtx, v3RestartRequest)
		require.NoError(t, err)

		assertEvent(t, rt.recorder, "Normal Restarted")
		assertNoEvent(t, rt.recorder)
	})

	t.Run("should be superseded by a newer restart", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		// set superseded to true, which means the restart is superseded by another restart - it is the return value
		// of the doguRestarter interface
		rt.expectApply(true, nil)
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded)
		assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded)
		assertEvent(t, rt.recorder, "Normal Superseded")
	})

	t.Run("should restart a stopped dogu without waiting for a rollout", func(t *testing.T) {
		dogu := v3Dogu(v3DoguReadyCondition)
		dogu.Spec.Stopped = true
		rt := newRestartV3Test(t, doguV3Restart(), dogu)
		rt.expectApply(false, nil)
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(nil)

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
		assertCondition(t, persisted, doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonDoguStopped)
	})

	t.Run("should retry if the annotation cannot be set", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		rt.expectApply(false, assert.AnError)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionFalse, doguv2.DoguRestartReasonPatchFailed)
		assert.False(t, restart.IsTerminal(persisted))
		assertEvent(t, rt.recorder, "Warning PatchFailed")
	})

	t.Run("should keep the restart and retry if the rollout cannot be checked", func(t *testing.T) {
		rt := newRestartV3Test(t, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		rt.expectApply(false, nil)
		rt.expectRolledOut(false, "", assert.AnError)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if workloads are rolled out")
		persisted := rt.persistedRestart(t)
		assertCondition(t, persisted, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)
		assert.Nil(t, meta.FindStatusCondition(persisted.Status.Conditions, doguv2.DoguRestartConditionReady))
		assert.False(t, restart.IsTerminal(persisted))
	})

	t.Run("should fail if the restart cannot be read", func(t *testing.T) {
		rt := newRestartV3TestWithFuncs(t, interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			},
		}, doguV3Restart())

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should fail if the dogu cannot be read", func(t *testing.T) {
		rt := newRestartV3TestWithFuncs(t, interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v3beta1.Dogu); ok {
					return assert.AnError
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}, doguV3Restart(), v3Dogu(v3DoguReadyCondition))

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to get dogu "test" of dogu restart "ecosystem/test-restart"`)
		assert.Empty(t, rt.persistedRestart(t).Status.Conditions)
	})

	t.Run("should fail if the garbage collection fails", func(t *testing.T) {
		rt := newRestartV3Test(t,
			doguV3Restart(condition(doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded),
				condition(doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded)),
			v3Dogu(v3DoguReadyCondition))
		rt.gc.EXPECT().DoGarbageCollection(testCtx, testDoguName).Return(assert.AnError)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to do garbage collection for dogu restart "ecosystem/test-restart"`)
	})

	t.Run("should fail if the conditions cannot be updated while the dogu is not idle", func(t *testing.T) {
		// the restarter mock fails the test if the dogu is restarted
		rt := newRestartV3TestWithFuncs(t, failStatusUpdate, doguV3Restart(), v3Dogu())

		result, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to update conditions of dogu restart "ecosystem/test-restart"`)
		// a failing reconciliation requeues with backoff, so no RequeueAfter is set
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("should return both errors if the restart and the condition update fail", func(t *testing.T) {
		rt := newRestartV3TestWithFuncs(t, failStatusUpdate, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		applyErr := errors.New("apply failed")
		rt.expectApply(false, applyErr)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, applyErr)
		require.ErrorIs(t, err, assert.AnError)
		assertEvent(t, rt.recorder, "Warning PatchFailed")
	})

	t.Run("should not emit the Restarted event if the Restarted condition cannot be updated", func(t *testing.T) {
		rt := newRestartV3TestWithFuncs(t, failStatusUpdate, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		rt.expectApply(false, nil)
		rt.expectRolledOut(false, "workloads not rolled out: Deployment/test", nil)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assertNoEvent(t, rt.recorder)
	})

	t.Run("should neither emit an event nor collect garbage if the final conditions cannot be updated", func(t *testing.T) {
		// the garbage collector mock fails the test if garbage is collected
		rt := newRestartV3TestWithFuncs(t, failStatusUpdate, doguV3Restart(), v3Dogu(v3DoguReadyCondition))
		rt.expectApply(true, nil)

		_, err := rt.sut.Reconcile(testCtx, v3RestartRequest)

		require.ErrorIs(t, err, assert.AnError)
		assertNoEvent(t, rt.recorder)
	})
}

func TestNewDoguRestartRequestMapper(t *testing.T) {
	t.Run("should map a workload to the v3 restarts of its dogu that wait for the rollout", func(t *testing.T) {
		running := doguV3Restart(condition(doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress))
		running.Name = "running"
		created := doguV3Restart()
		created.Name = "created"
		finished := doguV3Restart(condition(doguv2.DoguRestartConditionReady, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded))
		finished.Name = "finished"
		v2Restart := doguV3Restart()
		v2Restart.Name = "v2"
		v2Restart.Status.Phase = doguv2.RestartStatusPhaseStopping
		otherDogu := doguV3Restart()
		otherDogu.Name = "other"
		otherDogu.Spec.DoguName = "other"
		c := newRestartV3Client(running, created, finished, v2Restart, otherDogu)
		sut := newDoguRestartRequestMapper(c)
		workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name: testDoguName, Namespace: testNamespace, Labels: map[string]string{v3beta1.DoguLabelName: testDoguName},
		}}

		requests := sut(testCtx, workload)

		// new restarts are reconciled on creation, so only restarts waiting for the rollout are mapped
		assert.Equal(t, []reconcile.Request{
			{NamespacedName: client.ObjectKey{Namespace: testNamespace, Name: "running"}},
		}, requests)
	})

	t.Run("should not map a workload without dogu label", func(t *testing.T) {
		sut := newDoguRestartRequestMapper(newRestartV3Client(doguV3Restart(condition(doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress))))
		workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: testDoguName, Namespace: testNamespace}}

		requests := sut(testCtx, workload)

		assert.Empty(t, requests)
	})

	t.Run("should not map a workload if the restarts cannot be listed", func(t *testing.T) {
		c := newRestartV3ClientWithFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			},
		})
		sut := newDoguRestartRequestMapper(c)
		workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name: testDoguName, Namespace: testNamespace, Labels: map[string]string{v3beta1.DoguLabelName: testDoguName},
		}}

		requests := sut(testCtx, workload)

		assert.Empty(t, requests)
	})
}
