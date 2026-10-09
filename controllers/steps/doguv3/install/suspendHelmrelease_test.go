package install

import (
	"context"
	"errors"
	"testing"

	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestSuspendHelmReleaseStep_Run(t *testing.T) {
	tests := []suspendHelmReleaseRunCase{
		{
			name:       "continue when HelmRelease does not exist",
			wantResult: stepsv3.Continue(),
		},
		{
			name:            "suspend for paused Dogu",
			existingRelease: true,
			doguPaused:      true,
			wantResult:      stepsv3.RequeueAfter(0, ReasonReconciliationPaused, messageReconciliationPaused),
			wantSuspended:   true,
			wantCondition:   ReasonReconciliationPaused,
		},
		{
			name:            "suspend for stopped Dogu",
			existingRelease: true,
			doguStopped:     true,
			wantResult:      stepsv3.RequeueAfter(0, ReasonDoguStopped, messageDoguStopped),
			wantSuspended:   true,
			wantCondition:   ReasonDoguStopped,
		},
		{
			name:             "suspend for PVC resize",
			existingRelease:  true,
			resizeInProgress: true,
			wantResult:       stepsv3.RequeueAfter(0, ReasonPVCResizeInProgress, messagePVCResizeInProgress),
			wantSuspended:    true,
		},
		{
			name:             "resume when suspension reason has disappeared",
			existingRelease:  true,
			alreadySuspended: true,
			wantResult:       stepsv3.RequeueAfter(0, ReasonNotSuspended, messageNotSuspended),
			wantCondition:    ReasonNotSuspended,
		},
		{
			name:             "update condition when already suspended for a Dogu pause",
			existingRelease:  true,
			alreadySuspended: true,
			doguPaused:       true,
			wantResult:       stepsv3.Continue(),
			wantSuspended:    true,
			wantCondition:    ReasonReconciliationPaused,
		},
		{
			name:             "continue when already suspended for PVC resize",
			existingRelease:  true,
			alreadySuspended: true,
			resizeInProgress: true,
			wantResult:       stepsv3.Continue(),
			wantSuspended:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runSuspendHelmReleaseCase(t, test)
		})
	}
}

type suspendHelmReleaseRunCase struct {
	name             string
	existingRelease  bool
	alreadySuspended bool
	doguPaused       bool
	doguStopped      bool
	resizeInProgress bool
	wantResult       stepsv3.StepResult
	wantSuspended    bool
	wantCondition    string
}

func runSuspendHelmReleaseCase(t *testing.T, test suspendHelmReleaseRunCase) {
	t.Helper()
	dogu := newSuspendTestDogu()
	dogu.Spec.PauseReconciliation = test.doguPaused
	dogu.Spec.Stopped = test.doguStopped

	k8sClient := newSuspendRunTestClient(t, dogu, test.existingRelease, test.alreadySuspended)
	chartService := newSuspendRunChartService(t, dogu, test)
	checker := newSuspendRunResizeChecker(test.resizeInProgress)
	step := NewSuspendHelmReleaseStep(k8sClient, chartService, checker, nil)
	result := step.Run(t.Context(), dogu)

	assert.Equal(t, test.wantResult, result)
	assertSuspendRunResult(t, k8sClient, dogu, test)
}

func newSuspendRunTestClient(t *testing.T, dogu *doguv3.Dogu, existingRelease, alreadySuspended bool) client.Client {
	objects := []client.Object{dogu, newGlobalConfigForSuspendTest(dogu.Namespace)}
	if existingRelease {
		objects = append(objects, &flux.HelmRelease{
			Name: dogu.Spec.Name, Namespace: dogu.Namespace,
			Spec: flux.HelmReleaseSpec{Suspend: alreadySuspended},
		})
	}
	return newSuspendTestClient(t, objects...)
}

func newSuspendRunChartService(t *testing.T, dogu *doguv3.Dogu, test suspendHelmReleaseRunCase) *MockChartService {
	chartService := NewMockChartService(t)
	if !test.existingRelease {
		return chartService
	}
	chartService.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil)
	if !test.doguPaused && !test.doguStopped {
		chartService.EXPECT().Render(mock.Anything, dogu, mock.Anything).Return(nil, nil)
	}
	return chartService
}

func newSuspendRunResizeChecker(resizeInProgress bool) *suspendTestResizeChecker {
	checker := &suspendTestResizeChecker{}
	if resizeInProgress {
		checker.result = pvc.CheckResult{ResizeRequests: []pvc.ResizeRequest{{PVC: client.ObjectKey{Name: "data"}}}}
	}
	return checker
}

func assertSuspendRunResult(t *testing.T, k8sClient client.Client, dogu *doguv3.Dogu, test suspendHelmReleaseRunCase) {
	t.Helper()
	release := &flux.HelmRelease{}
	err := k8sClient.Get(t.Context(), client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, release)
	if !test.existingRelease {
		assert.True(t, apierrors.IsNotFound(err))
		return
	}
	require.NoError(t, err)
	assert.Equal(t, test.wantSuspended, release.Spec.Suspend)
	if test.wantCondition != "" {
		assertSuspendTestCondition(t, k8sClient, dogu, test.wantCondition)
	}
}

func TestSuspendHelmReleaseStep_RunErrors(t *testing.T) {
	t.Run("requeue when reading HelmRelease fails", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		k8sClient := fake.NewClientBuilder().
			WithScheme(newSuspendTestScheme(t)).
			WithObjects(newGlobalConfigForSuspendTest(dogu.Namespace)).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return errors.New("get failed")
				},
			}).
			Build()

		step := NewSuspendHelmReleaseStep(k8sClient, nil, nil, nil)
		result := step.Run(t.Context(), dogu)

		require.Error(t, result.Err)
		assert.Equal(t, doguv3.ReasonInstalling, result.ReadyReason)
	})

	t.Run("requeue when assembling values fails", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		k8sClient := newSuspendTestClient(t,
			newGlobalConfigForSuspendTest(dogu.Namespace),
			&flux.HelmRelease{Name: dogu.Spec.Name, Namespace: dogu.Namespace},
		)
		chartService := NewMockChartService(t)
		chartService.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, errors.New("metadata failed"))

		result := NewSuspendHelmReleaseStep(k8sClient, chartService, &suspendTestResizeChecker{}, nil).Run(t.Context(), dogu)

		require.Error(t, result.Err)
		assert.Equal(t, doguv3.ReasonReconciliationPaused, result.ReadyReason)
	})

	t.Run("requeue when checking PVC resize fails", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		k8sClient := newSuspendTestClient(t,
			newGlobalConfigForSuspendTest(dogu.Namespace),
			&flux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: dogu.Spec.Name, Namespace: dogu.Namespace}},
		)
		chartService := NewMockChartService(t)
		chartService.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil)
		chartService.EXPECT().Render(mock.Anything, dogu, mock.Anything).Return(nil, nil)
		checker := &suspendTestResizeChecker{err: errors.New("resize check failed")}

		result := NewSuspendHelmReleaseStep(k8sClient, chartService, checker, nil).Run(t.Context(), dogu)

		require.Error(t, result.Err)
		assert.Equal(t, doguv3.ReasonReconciliationPaused, result.ReadyReason)
	})
}

func TestSuspendHelmReleaseStep_existingHelmRelease(t *testing.T) {
	t.Run("returns nil when HelmRelease is not found", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		step := NewSuspendHelmReleaseStep(newSuspendTestClient(t), nil, nil, nil)

		release, err := step.existingHelmRelease(t.Context(), dogu)

		require.NoError(t, err)
		assert.Nil(t, release)
	})

	t.Run("returns the existing HelmRelease", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		expected := &flux.HelmRelease{Name: dogu.Spec.Name, Namespace: dogu.Namespace}
		step := NewSuspendHelmReleaseStep(newSuspendTestClient(t, expected), nil, nil, nil)

		release, err := step.existingHelmRelease(t.Context(), dogu)

		require.NoError(t, err)
		assert.Equal(t, expected, release)
	})

	t.Run("wraps client errors", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		clientWithError := fake.NewClientBuilder().
			WithScheme(newSuspendTestScheme(t)).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return errors.New("get failed")
				},
			}).
			Build()
		step := NewSuspendHelmReleaseStep(clientWithError, nil, nil, nil)

		release, err := step.existingHelmRelease(t.Context(), dogu)

		require.ErrorContains(t, err, "failed to get HelmRelease")
		assert.NotNil(t, release)
	})
}

func TestSuspendHelmReleaseStep_ChangeSuspendValueOfHelmRelease(t *testing.T) {
	t.Run("updates the suspend value", func(t *testing.T) {
		tests := []struct {
			name    string
			initial bool
			desired bool
		}{
			{name: "sets suspend", initial: false, desired: true},
			{name: "clears suspend", initial: true, desired: false},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				dogu := newSuspendTestDogu()
				release := &flux.HelmRelease{
					Name: dogu.Spec.Name, Namespace: dogu.Namespace,
					Spec: flux.HelmReleaseSpec{Suspend: test.initial},
				}
				k8sClient := newSuspendTestClient(t, release)
				step := NewSuspendHelmReleaseStep(k8sClient, nil, nil, nil)

				err := step.ChangeSuspendValueOfHelmRelease(t.Context(), release, test.desired)

				require.NoError(t, err)
				updated := &flux.HelmRelease{}
				require.NoError(t, k8sClient.Get(t.Context(), client.ObjectKeyFromObject(release), updated))
				assert.Equal(t, test.desired, updated.Spec.Suspend)
				assert.Equal(t, test.initial, release.Spec.Suspend)
			})
		}
	})

	t.Run("wraps patch errors", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		clientWithError := fake.NewClientBuilder().
			WithScheme(newSuspendTestScheme(t)).
			WithObjects(&flux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: dogu.Spec.Name, Namespace: dogu.Namespace}}).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
					return errors.New("patch failed")
				},
			}).
			Build()
		step := NewSuspendHelmReleaseStep(clientWithError, nil, nil, nil)

		err := step.ChangeSuspendValueOfHelmRelease(t.Context(), &flux.HelmRelease{
			Name: dogu.Spec.Name, Namespace: dogu.Namespace,
		}, true)

		require.ErrorContains(t, err, "failed to suspend HelmRelease")
	})
}

func TestSuspendHelmReleaseStep_suspensionReason(t *testing.T) {
	tests := []struct {
		name          string
		pause         bool
		stopped       bool
		checkResult   pvc.CheckResult
		checkError    error
		renderError   error
		values        *apiext.JSON
		wantReason    string
		wantMessage   string
		wantErrorText string
	}{
		{
			name:        "pause takes precedence",
			pause:       true,
			stopped:     true,
			wantReason:  ReasonReconciliationPaused,
			wantMessage: messageReconciliationPaused,
		},
		{
			name:        "stopped Dogu",
			stopped:     true,
			wantReason:  ReasonDoguStopped,
			wantMessage: messageDoguStopped,
		},
		{
			name:        "PVC resize in progress",
			checkResult: pvc.CheckResult{ResizeRequests: []pvc.ResizeRequest{{PVC: client.ObjectKey{Name: "data"}}}},
			wantReason:  ReasonPVCResizeInProgress,
			wantMessage: messagePVCResizeInProgress,
		},
		{
			name:        "no suspension reason",
			wantReason:  ReasonNotSuspended,
			wantMessage: messageNotSuspended,
		},
		{
			name:          "invalid JSON values",
			values:        &apiext.JSON{Raw: []byte("{")},
			wantErrorText: "failed to decode assembled chart values",
		},
		{
			name:          "chart rendering fails",
			renderError:   errors.New("render failed"),
			wantErrorText: "failed to render chart",
		},
		{
			name:          "PVC resize check fails",
			checkError:    errors.New("check failed"),
			wantErrorText: "failed to check PVC resize requirements",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dogu := newSuspendTestDogu()
			dogu.Spec.PauseReconciliation = test.pause
			dogu.Spec.Stopped = test.stopped
			values := test.values
			if values == nil {
				values = &apiext.JSON{Raw: []byte(`{}`)}
			}

			chartService := NewMockChartService(t)
			checker := &suspendTestResizeChecker{result: test.checkResult, err: test.checkError}
			if !test.pause && !test.stopped && test.wantErrorText != "failed to decode assembled chart values" {
				chartService.EXPECT().Render(mock.Anything, dogu, map[string]any{}).Return(nil, test.renderError)
			}
			step := NewSuspendHelmReleaseStep(nil, chartService, checker, nil)

			reason, message, err := step.suspensionReason(t.Context(), dogu, values)

			if test.wantErrorText != "" {
				require.ErrorContains(t, err, test.wantErrorText)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantReason, reason)
			assert.Equal(t, test.wantMessage, message)
			if !test.pause && !test.stopped {
				assert.Equal(t, dogu.Namespace, checker.namespace)
			}
		})
	}
}

func TestSuspendHelmReleaseStep_setSuspendedConditionOnDoguCR(t *testing.T) {
	tests := []struct {
		name    string
		reason  string
		message string
		status  metav1.ConditionStatus
	}{
		{
			name:    "sets true condition for a suspension reason",
			reason:  ReasonDoguStopped,
			message: messageDoguStopped,
			status:  metav1.ConditionTrue,
		},
		{
			name:    "sets false condition when no reason remains",
			reason:  ReasonNotSuspended,
			message: messageNotSuspended,
			status:  metav1.ConditionFalse,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dogu := newSuspendTestDogu()
			k8sClient := newSuspendTestClient(t, dogu)
			step := NewSuspendHelmReleaseStep(k8sClient, nil, nil, nil)

			err := step.setSuspendedConditionOnDoguCR(t.Context(), dogu, test.reason, test.message)

			require.NoError(t, err)
			assertSuspendTestConditionWithStatus(t, k8sClient, dogu, test.reason, test.status, test.message)
		})
	}

	t.Run("wraps status update errors", func(t *testing.T) {
		dogu := newSuspendTestDogu()
		k8sClient := fake.NewClientBuilder().
			WithScheme(newSuspendTestScheme(t)).
			WithObjects(dogu).
			WithStatusSubresource(&doguv3.Dogu{}).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
					return errors.New("status update failed")
				},
			}).
			Build()
		step := NewSuspendHelmReleaseStep(k8sClient, nil, nil, nil)

		err := step.setSuspendedConditionOnDoguCR(t.Context(), dogu, ReasonDoguStopped, messageDoguStopped)

		require.ErrorContains(t, err, "failed to update Dogu suspended condition")
	})
}

type suspendTestResizeChecker struct {
	result    pvc.CheckResult
	err       error
	namespace string
}

func (checker *suspendTestResizeChecker) Check(_ context.Context, namespace string, _ []*unstructured.Unstructured) (pvc.CheckResult, error) {
	checker.namespace = namespace
	return checker.result, checker.err
}

func newSuspendTestDogu() *doguv3.Dogu {
	return &doguv3.Dogu{
		Name: "dogu-resource", Namespace: "dogu-operator", Generation: 3,
		Spec: doguv3.DoguSpec{
			Name:          "my-dogu",
			DoguNamespace: "dogu-space",
		},
	}
}

func newGlobalConfigForSuspendTest(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		Namespace: namespace, Name: "global-config",
		Data: map[string]string{"config.yaml": "{}"},
	}
}

func newSuspendTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, doguv3.AddToScheme(scheme))
	require.NoError(t, flux.AddToScheme(scheme))
	return scheme
}

func newSuspendTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newSuspendTestScheme(t)).
		WithObjects(objects...).
		WithStatusSubresource(&doguv3.Dogu{}).
		Build()
}

func assertSuspendTestCondition(t *testing.T, k8sClient client.Client, dogu *doguv3.Dogu, reason string) {
	t.Helper()
	status := metav1.ConditionTrue
	message := messageDoguStopped
	if reason == ReasonReconciliationPaused {
		message = messageReconciliationPaused
	} else if reason == ReasonNotSuspended {
		status = metav1.ConditionFalse
		message = messageNotSuspended
	}
	assertSuspendTestConditionWithStatus(t, k8sClient, dogu, reason, status, message)
}

func assertSuspendTestConditionWithStatus(t *testing.T, k8sClient client.Client, dogu *doguv3.Dogu, reason string, status metav1.ConditionStatus, message string) {
	t.Helper()
	updated := &doguv3.Dogu{}
	require.NoError(t, k8sClient.Get(t.Context(), client.ObjectKeyFromObject(dogu), updated))
	condition := apimeta.FindStatusCondition(updated.Status.Conditions, ConditionSuspended)
	require.NotNil(t, condition)
	assert.Equal(t, status, condition.Status)
	assert.Equal(t, reason, condition.Reason)
	assert.Equal(t, message, condition.Message)
	assert.Equal(t, dogu.Generation, condition.ObservedGeneration)
}

var _ pvc.ResizeChecker = (*suspendTestResizeChecker)(nil)
