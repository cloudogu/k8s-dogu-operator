package install

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	v3steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_combineValues(t *testing.T) {
	t.Run("should combine values from without error", func(t *testing.T) {
		// given
		inputDogu := &doguv3.Dogu{}
		assembledDoguValues := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		valueAsmMock := NewMockValueAssembler(t)
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := NewMockChartService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(doguMetadataValues, true, nil)
		valueAsmMock.EXPECT().Assemble(testCtx, inputDogu, doguMetadataValues).Return(assembledDoguValues, nil)

		// when
		actual, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.NoError(t, err)
		expectedVal := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		expectedBytes, err := json.Marshal(expectedVal)
		expectedJson := apiext.JSON{Raw: expectedBytes}
		assert.Equal(t, expectedJson, *actual)
	})

	t.Run("should fail on dogu metadata value service retrieval", func(t *testing.T) {
		// given
		inputDogu := &doguv3.Dogu{}
		valueAsmMock := NewMockValueAssembler(t)
		doguValuesMetadataSvcMock := NewMockChartService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(nil, false, assert.AnError)

		// when
		_, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to retrieve dogu metadata values: ")
	})

	t.Run("should fail on dogu value assembly", func(t *testing.T) {
		// given
		inputDogu := &doguv3.Dogu{}
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := NewMockChartService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(doguMetadataValues, true, nil)
		valueAsmMock := NewMockValueAssembler(t)
		valueAsmMock.EXPECT().Assemble(testCtx, inputDogu, doguMetadataValues).Return(nil, assert.AnError)

		// when
		_, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to assemble dogu values: ")
	})
}

func Test_configureHelmRelease(t *testing.T) {
	t.Run("configure new helm release", func(t *testing.T) {
		release := &fluxhelm.HelmRelease{
			Name:      "dogu-release",
			Namespace: "namespace",
		}
		dogu := &doguv3.Dogu{
			Name:      "dogu",
			Namespace: "namespace",
			Spec: doguv3.DoguSpec{
				Name:          "dogu",
				DoguNamespace: "namespace",
				Version:       "1.2.3",
			},
		}
		values := &apiext.JSON{}
		retryInterval := &metav1.Duration{Duration: 30 * time.Second}
		reconcileInterval := &metav1.Duration{Duration: 60 * time.Second}

		configureHelmRelease(release, dogu, values, retryInterval, reconcileInterval)

		checkExpectedValues(t, release, retryInterval, reconcileInterval, values)

		// status should not be set
		assert.Empty(t, release.Status)
	})
	t.Run("configure existing helm release", func(t *testing.T) {
		release := &fluxhelm.HelmRelease{
			Name:      "dogu-release",
			Namespace: "namespace",
			Labels: map[string]string{
				"my.label": "my.value",
			},
			Status: fluxhelm.HelmReleaseStatus{
				ObservedGeneration: 1,
				HelmChart:          "oci://helm/chart",
				Failures:           5,
			},
			Spec: fluxhelm.HelmReleaseSpec{
				ChartRef: &fluxhelm.CrossNamespaceSourceReference{
					APIVersion: "v7",
					Kind:       "FunnyKind",
					Name:       "SomeName",
					Namespace:  "other-namespace",
				},
				Interval:    metav1.Duration{Duration: 7 * time.Hour},
				ReleaseName: "some-release",
				DriftDetection: &fluxhelm.DriftDetection{
					Mode: fluxhelm.DriftDetectionDisabled,
					Ignore: []fluxhelm.IgnoreRule{{
						Paths: []string{"path"},
						Target: &kustomize.Selector{
							AnnotationSelector: "app=ces",
						},
					}},
				},
				CommonMetadata: nil,
			},
		}
		dogu := &doguv3.Dogu{
			Name:      "dogu",
			Namespace: "namespace",
			Spec: doguv3.DoguSpec{
				Name:          "dogu",
				DoguNamespace: "namespace",
				Version:       "1.2.3",
			},
		}
		values := &apiext.JSON{}
		retryInterval := &metav1.Duration{Duration: 30 * time.Second}
		reconcileInterval := &metav1.Duration{Duration: 60 * time.Second}

		configureHelmRelease(release, dogu, values, retryInterval, reconcileInterval)

		checkExpectedValues(t, release, retryInterval, reconcileInterval, values)

		// existing label should be retained
		assert.Equal(t, "my.value", release.Labels["my.label"])

		// existing status should be unchanged
		originalStatus := fluxhelm.HelmReleaseStatus{
			ObservedGeneration: 1,
			HelmChart:          "oci://helm/chart",
			Failures:           5,
		}
		assert.Equal(t, originalStatus, release.Status)
	})
	t.Run("propagate schema validation setting and re-enable it", func(t *testing.T) {
		release := &fluxhelm.HelmRelease{}
		dogu := &doguv3.Dogu{}
		values := &apiext.JSON{}
		retryInterval := &metav1.Duration{Duration: 30 * time.Second}
		reconcileInterval := &metav1.Duration{Duration: 60 * time.Second}

		for _, skipValidation := range []bool{false, true, false} {
			dogu.Spec.SkipSchemaValidation = skipValidation

			configureHelmRelease(release, dogu, values, retryInterval, reconcileInterval)

			assert.Equal(t, skipValidation, release.Spec.Install.DisableSchemaValidation, "install")
			assert.Equal(t, skipValidation, release.Spec.Upgrade.DisableSchemaValidation, "upgrade")
		}
	})
}

func checkExpectedValues(t *testing.T, release *fluxhelm.HelmRelease, retryInterval *metav1.Duration, reconcileInterval *metav1.Duration, values *apiext.JSON) {
	assert.Equal(t, "dogu-release", release.Name)
	assert.Equal(t, "namespace", release.Namespace)
	assert.Empty(t, release.Spec.ReleaseName)
	assert.Equal(t, "dogu", release.Spec.ChartRef.Name)
	assert.Equal(t, "namespace", release.Spec.ChartRef.Namespace)
	assert.Equal(t, "OCIRepository", release.Spec.ChartRef.Kind)
	assert.Equal(t, "RetryOnFailure", release.Spec.Install.Strategy.Name)
	assert.Equal(t, retryInterval, release.Spec.Install.Strategy.RetryInterval)
	assert.Equal(t, "RetryOnFailure", release.Spec.Upgrade.Strategy.Name)
	assert.Equal(t, retryInterval, release.Spec.Upgrade.Strategy.RetryInterval)
	assert.Equal(t, *reconcileInterval, release.Spec.Interval)
	assert.Equal(t, values, release.Spec.Values)
	assert.Equal(t, fluxhelm.DriftDetectionEnabled, release.Spec.DriftDetection.Mode)
	assert.Empty(t, release.Spec.DriftDetection.Ignore)

	assert.Equal(t, "ces", release.Labels["sharding.fluxcd.io/key"])
	assert.Equal(t, "dogu", release.Labels[doguv3.DoguLabelName])
	assert.Equal(t, "1.2.3", release.Labels[doguv3.DoguLabelVersion])

	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.LegacyLabelKeyApp])
	assert.Equal(t, "dogu", release.Spec.CommonMetadata.Labels[doguv2.DoguLabelName])

	assert.Equal(t, "dogu", release.Spec.CommonMetadata.Labels[resource.CommonLabelKeyName])
	assert.Equal(t, "1.2.3", release.Spec.CommonMetadata.Labels[resource.CommonLabelKeyVersion])
	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.CommonLabelKeyPartOf])
	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.CloudoguLabelKeyApp])
}

func TestNewEnsureHelmReleaseStep(t *testing.T) {
	k8sClient := NewMockK8sClient(t)
	operatorConfig := &config.OperatorConfig{
		DoguHelmRetryInterval:          1 * time.Second,
		DoguHelmReconciliationInterval: 2 * time.Minute,
	}
	valueService := NewMockChartService(t)
	recorder := NewMockEventRecorder(t)

	step := NewEnsureHelmReleaseStep(k8sClient, operatorConfig, valueService, recorder)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, valueService, step.chartService)
	assert.Equal(t, recorder, step.eventRecorder)
	assert.Equal(t, metav1.Duration{Duration: 1 * time.Second}, step.retryInterval)
	assert.Equal(t, metav1.Duration{Duration: 2 * time.Minute}, step.helmReconcileInterval)
}

func TestEnsureHelmReleaseStep_Run(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = core.AddToScheme(testScheme)
	_ = doguv3.AddToScheme(testScheme)
	_ = fluxhelm.AddToScheme(testScheme)
	doguResource := &doguv3.Dogu{
		Name:      "dogu",
		Namespace: "namespace",
		Spec: doguv3.DoguSpec{
			Name:          "dogu",
			DoguNamespace: "namespace",
			Version:       "1.2.3",
		},
	}
	globalConfig := core.ConfigMap{
		Namespace: "namespace",
		Name:      "global-config",
		Data: map[string]string{
			"config.yaml": "{}",
		},
	}

	tests := []struct {
		name       string
		setupMocks func(*testing.T) (K8sClient, ChartService, EventRecorder)
		want       v3steps.StepResult
	}{
		{
			name: "should retry on failing value assembling",
			setupMocks: func(*testing.T) (K8sClient, ChartService, EventRecorder) {
				k8sClient := NewMockK8sClient(t)
				k8sClient.EXPECT().
					Get(mock.Anything, mock.Anything, mock.Anything).
					Return(apierrors.NewNotFound(fluxhelm.GroupVersion.WithResource("helmreleases").GroupResource(), doguResource.Spec.Name))
				metaValueService := NewMockChartService(t)
				metaValueService.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, assert.AnError)
				return k8sClient, metaValueService, nil
			},
			want: v3steps.StepResult{
				Err:          fmt.Errorf("failed to retrieve dogu metadata values: %w", assert.AnError),
				ReadyReason:  "Installing",
				ReadyMessage: "failed to retrieve dogu metadata values: assert.AnError general error for testing",
				Continue:     false,
			},
		},
		{
			name: "should retry on failing create or patch",
			setupMocks: func(*testing.T) (K8sClient, ChartService, EventRecorder) {
				fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig).Build()
				k8sClient := NewMockK8sClient(t)
				k8sClient.EXPECT().Scheme().Return(testScheme)
				k8sClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return fakeClient.Get(ctx, key, obj)
				})
				k8sClient.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

				service := NewMockChartService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				return k8sClient, service, nil
			},
			want: v3steps.StepResult{
				Err:          fmt.Errorf("failed to createOrPatch HelmRelease \"dogu\": %w", assert.AnError),
				ReadyReason:  "Installing",
				ReadyMessage: "failed to createOrPatch HelmRelease \"dogu\": assert.AnError general error for testing",
				Continue:     false,
			},
		},
		{
			name: "should continue on successful creation of helm release",
			setupMocks: func(t *testing.T) (K8sClient, ChartService, EventRecorder) {
				k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig).Build()
				service := NewMockChartService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, core.EventTypeNormal, doguv3.ConditionChartAvailable, "HelmRelease created")
				return k8sClient, service, recorder
			},
			want: v3steps.Continue(),
		},
		{
			name: "should continue on successful update of existing helm release",
			setupMocks: func(t *testing.T) (K8sClient, ChartService, EventRecorder) {
				release := fluxhelm.HelmRelease{
					Name:      "dogu",
					Namespace: "namespace",
				}
				k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig, &release).Build()
				service := NewMockChartService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, core.EventTypeNormal, doguv3.ConditionChartAvailable, "HelmRelease updated")
				return k8sClient, service, recorder
			},
			want: v3steps.Continue(),
		},
		{
			name: "should abort when existing helm release is suspended",
			setupMocks: func(t *testing.T) (K8sClient, ChartService, EventRecorder) {
				release := fluxhelm.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dogu",
						Namespace: "namespace",
					},
					Spec: fluxhelm.HelmReleaseSpec{
						Suspend: true,
					},
				}
				k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig, &release).Build()
				return k8sClient, NewMockChartService(t), nil
			},
			want: v3steps.Abort(ReasonReconciliationPaused, messageSuspended),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k8sClient, metaValueService, recorder := test.setupMocks(t)
			step := &EnsureHelmReleaseStep{k8sClient: k8sClient, chartService: metaValueService, eventRecorder: recorder}

			result := step.Run(t.Context(), doguResource)

			assert.Equal(t, test.want, result)
		})
	}
}
