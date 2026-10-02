package install

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_combineValues(t *testing.T) {
	t.Run("should combine values from without error", func(t *testing.T) {
		// given
		inputDogu := &v3beta1.Dogu{}
		assembledDoguValues := values3.Values{"foo": "bar", "fqdn": "example.invalid", "logging": "error"}
		valueAsmMock := newMockDoguPatchAssembler(t)
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
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
		inputDogu := &v3beta1.Dogu{}
		valueAsmMock := newMockDoguPatchAssembler(t)
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(nil, false, assert.AnError)

		// when
		_, err := combineValues(testCtx, inputDogu, doguValuesMetadataSvcMock, valueAsmMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to retrieve dogu metadata values: ")
	})

	t.Run("should fail on dogu value assembly", func(t *testing.T) {
		// given
		inputDogu := &v3beta1.Dogu{}
		var doguMetadataValues []byte
		doguValuesMetadataSvcMock := newMockDoguMetadataValueService(t)
		doguValuesMetadataSvcMock.EXPECT().DoguMetaValues(testCtx, inputDogu).Return(doguMetadataValues, true, nil)
		valueAsmMock := newMockDoguPatchAssembler(t)
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
		dogu := &v3beta1.Dogu{
			Name:      "dogu",
			Namespace: "namespace",
			Spec: v3beta1.DoguSpec{
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
		dogu := &v3beta1.Dogu{
			Name:      "dogu",
			Namespace: "namespace",
			Spec: v3beta1.DoguSpec{
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
	assert.Equal(t, "dogu", release.Labels[v3beta1.DoguLabelName])
	assert.Equal(t, "1.2.3", release.Labels[v3beta1.DoguLabelVersion])

	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.LabelKeyApp])
	assert.Equal(t, "dogu", release.Spec.CommonMetadata.Labels[doguv2.DoguLabelName])

	assert.Equal(t, "dogu", release.Spec.CommonMetadata.Labels[resource.LabelKeyAppKubernetesIoName])
	assert.Equal(t, "1.2.3", release.Spec.CommonMetadata.Labels[resource.LabelKeyAppKubernetesIoVersion])
	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.LabelKeyAppKubernetesIoPartOf])
	assert.Equal(t, "ces", release.Spec.CommonMetadata.Labels[resource.LabelKeyK8sCloudoguComApp])
}

func TestNewEnsureHelmReleaseStep(t *testing.T) {
	k8sClient := NewMockK8sClient(t)
	operatorConfig := &config.OperatorConfig{
		DoguHelmRetryInterval:          1 * time.Second,
		DoguHelmReconciliationInterval: 2 * time.Minute,
	}
	valueService := newMockDoguMetadataValueService(t)
	recorder := NewMockEventRecorder(t)

	step := NewEnsureHelmReleaseStep(k8sClient, operatorConfig, valueService, recorder)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, valueService, step.doguMetadataValueSvc)
	assert.Equal(t, recorder, step.eventRecorder)
	assert.Equal(t, metav1.Duration{Duration: 1 * time.Second}, step.retryInterval)
	assert.Equal(t, metav1.Duration{Duration: 2 * time.Minute}, step.helmReconcileInterval)
}

func TestEnsureHelmReleaseStep_Run(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = core.AddToScheme(testScheme)
	_ = v3beta1.AddToScheme(testScheme)
	_ = fluxhelm.AddToScheme(testScheme)
	doguResource := &v3beta1.Dogu{
		Name:      "dogu",
		Namespace: "namespace",
		Spec: v3beta1.DoguSpec{
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
		setupMocks func(*testing.T) (K8sClient, DoguValuesMetadataService, EventRecorder)
		want       doguv3.StepResult
	}{
		{
			name: "should retry on failing value assembling",
			setupMocks: func(*testing.T) (K8sClient, DoguValuesMetadataService, EventRecorder) {
				metaValueService := newMockDoguMetadataValueService(t)
				metaValueService.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, assert.AnError)
				return nil, metaValueService, nil
			},
			want: doguv3.StepResult{
				Err:          fmt.Errorf("failed to retrieve dogu metadata values: %w", assert.AnError),
				ReadyReason:  "Installing",
				ReadyMessage: "failed to retrieve dogu metadata values: assert.AnError general error for testing",
				Continue:     false,
			},
		},
		{
			name: "should retry on failing create or patch",
			setupMocks: func(*testing.T) (K8sClient, DoguValuesMetadataService, EventRecorder) {
				k8sClient := createFailingClient{
					delegate: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig).Build(),
				}

				service := newMockDoguMetadataValueService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				return k8sClient, service, nil
			},
			want: doguv3.StepResult{
				Err:          fmt.Errorf("failed to createOrPatch HelmRelease \"dogu\": %w", assert.AnError),
				ReadyReason:  "Installing",
				ReadyMessage: "failed to createOrPatch HelmRelease \"dogu\": assert.AnError general error for testing",
				Continue:     false,
			},
		},
		{
			name: "should continue on successful creation of helm release",
			setupMocks: func(t *testing.T) (K8sClient, DoguValuesMetadataService, EventRecorder) {
				k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig).Build()
				service := newMockDoguMetadataValueService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, core.EventTypeNormal, v3beta1.ConditionChartAvailable, "HelmRelease created")
				return k8sClient, service, recorder
			},
			want: doguv3.Continue(),
		},
		{
			name: "should continue on successful update of existing helm release",
			setupMocks: func(t *testing.T) (K8sClient, DoguValuesMetadataService, EventRecorder) {
				release := fluxhelm.HelmRelease{
					Name:      "dogu",
					Namespace: "namespace",
				}
				k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&globalConfig, &release).Build()
				service := newMockDoguMetadataValueService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, doguResource).Return(nil, false, nil)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, core.EventTypeNormal, v3beta1.ConditionChartAvailable, "HelmRelease updated")
				return k8sClient, service, recorder
			},
			want: doguv3.Continue(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k8sClient, metaValueService, recorder := test.setupMocks(t)
			step := &EnsureHelmReleaseStep{k8sClient: k8sClient, doguMetadataValueSvc: metaValueService, eventRecorder: recorder}

			result := step.Run(t.Context(), doguResource)

			assert.Equal(t, test.want, result)
		})
	}
}

type createFailingClient struct {
	client.Client
	delegate client.Client
}

func (c createFailingClient) Scheme() *runtime.Scheme {
	return c.delegate.Scheme()
}

func (c createFailingClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	return assert.AnError
}

func (c createFailingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return c.delegate.Get(ctx, key, obj, opts...)
}
