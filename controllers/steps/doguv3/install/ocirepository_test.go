package install

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	dogulibv3 "github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/dogu-lib/doguv3/doguregistry"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace     = "ecosystem"
	testVersion       = "1.0.0"
	testDoguName      = "nexus"
	testDoguNamespace = "testing"
	testChartURL      = "oci://localhost:5000/testing/nexus"
)

var (
	testCtx        = context.Background()
	testIdentifier = dogulibv3.Identifier{
		DoguNamespace: testDoguNamespace,
		Name:          testDoguName,
		Version:       testVersion,
	}
	testDoguDescriptor = &dogulibv3.Dogu{
		Chart:   testChartURL,
		Version: testVersion,
	}
	testDoguResource = &v3beta1.Dogu{
		Name:      testDoguName,
		Namespace: testNamespace,
		Spec: v3beta1.DoguSpec{
			Name:          testDoguName,
			DoguNamespace: testDoguNamespace,
			Version:       testVersion,
		},
		Status: v3beta1.DoguStatus{
			Conditions: []metav1.Condition{},
		},
	}
	testNamespacedName = types.NamespacedName{
		Namespace: testNamespace,
		Name:      testDoguName,
	}
	testRepository = flux.OCIRepository{
		Namespace: testNamespace,
		Name:      testDoguName,
		Spec: flux.OCIRepositorySpec{
			URL: testChartURL,
			Reference: &flux.OCIRepositoryRef{
				Tag: testVersion,
			},
		},
	}
	testStepResultContinue = doguv3.StepResult{Continue: true}
	testScheme             = runtime.NewScheme()
	_                      = v3beta1.AddToScheme(testScheme)
	_                      = flux.AddToScheme(testScheme)
)

func TestEnsureOCIRepositoryStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu)
		want     doguv3.StepResult
		assertFn func(t *testing.T, client K8sClient)
	}{
		{
			name: "should continue on success and create repository if not existent",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorderMock := NewMockEventRecorder(t)
				recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository created")

				return c, doguRegistryMock, recorderMock, doguResource
			},
			want:     testStepResultContinue,
			assertFn: checkExpectedRepositoryValues,
		},
		{
			name: "should continue on success and update if repository already exists",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				existingRepository := testRepository.DeepCopy()
				existingRepository.Spec.Reference.Tag = "0.9.0"
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorderMock := NewMockEventRecorder(t)
				recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository updated")

				return c, doguRegistryMock, recorderMock, doguResource
			},
			want:     testStepResultContinue,
			assertFn: checkExpectedRepositoryValues,
		},
		{
			name: "should abort on unrecoverable dogu registry error",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				registryErr := doguregistry.NewGenericError(assert.AnError)
				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(nil, registryErr)

				return nil, doguRegistryMock, nil, testDoguResource.DeepCopy()
			},
			want: func() doguv3.StepResult {
				registryErr := doguregistry.NewGenericError(assert.AnError)
				wrappedErr := fmt.Errorf("failed to get dogu descriptor for identifier %v: %w", testIdentifier, registryErr)
				return doguv3.Abort(v3beta1.ReasonDownloadFailed, wrappedErr.Error())
			}(),
		},
		{
			name: "should requeue with error on recoverable dogu registry error",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				connectionErr := doguregistry.NewConnectionError(assert.AnError)
				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(nil, connectionErr)

				return nil, doguRegistryMock, nil, testDoguResource.DeepCopy()
			},
			want: func() doguv3.StepResult {
				connectionErr := doguregistry.NewConnectionError(assert.AnError)
				wrappedErr := fmt.Errorf("failed to get dogu descriptor for identifier %v: %w", testIdentifier, connectionErr)
				return doguv3.RequeueWithError(wrappedErr, v3beta1.ReasonInstalling)
			}(),
		},
		{
			name: "should requeue with error on error patching repository",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				existingRepository := testRepository.DeepCopy()
				existingRepository.Spec.Reference.Tag = "0.9.0"
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return assert.AnError
					},
				}).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				return c, doguRegistryMock, nil, doguResource
			},
			want: func() doguv3.StepResult {
				wrappedErr := fmt.Errorf("failed to createOrPatch OCIRepository %q: %w", testDoguResource.Spec.Name, assert.AnError)
				return doguv3.RequeueWithError(wrappedErr, v3beta1.ReasonInstalling)
			}(),
		},
		{
			name: "Should skip creating OCIRepository, if PauseReconciliation is set",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.PauseReconciliation = true
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(doguResource).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionUpdatePending, "Changes not propagated to OCIRepository, because dogu reconciliation is paused")

				return c, doguRegistryMock, recorder, doguResource
			},
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				repo := &flux.OCIRepository{}
				err := client.Get(t.Context(), types.NamespacedName{Name: testDoguResource.Name, Namespace: testDoguResource.Namespace}, repo)
				assert.True(t, errors.IsNotFound(err))
			},
		},
		{
			name: "Should skip updating OCIRepository, if PauseReconciliation is set and nothing changed in the dogu",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.PauseReconciliation = true
				existingRepo := testRepository.DeepCopy()
				existingRepo.Spec.Provider = flux.GenericOCIProvider
				existingRepo.Spec.Timeout = &metav1.Duration{Duration: 60 * time.Second}
				existingRepo.Spec.SecretRef = &meta.LocalObjectReference{Name: containerRegistrySecret}
				existingRepo.Spec.Interval = metav1.Duration{Duration: time.Hour * 6}
				existingRepo.Labels = map[string]string{
					labelKeyFluxSharding:     resource.LabelValueCes,
					v3beta1.DoguLabelName:    testDoguName,
					v3beta1.DoguLabelVersion: testVersion,
					"custom.label":           "anything",
				}

				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepo).WithStatusSubresource(doguResource).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorder := NewMockEventRecorder(t)

				return c, doguRegistryMock, recorder, doguResource
			},
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				repo := &flux.OCIRepository{}
				err := client.Get(t.Context(), types.NamespacedName{Name: testDoguResource.Name, Namespace: testDoguResource.Namespace}, repo)
				assert.NoError(t, err)
				// spec should be unchanged
				assert.Equal(t, testChartURL, repo.Spec.URL)
				assert.Equal(t, testVersion, repo.Spec.Reference.Tag)
			},
		},
		{
			name: "Should skip updating OCIRepository, if PauseReconciliation is set, but dogu version changed",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Spec.PauseReconciliation = true
				existingRepo := testRepository.DeepCopy()
				existingRepo.Spec.Reference.Tag = "1.2.3-different"
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepo).WithStatusSubresource(doguResource).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionUpdatePending, "Changes not propagated to OCIRepository, because dogu reconciliation is paused")

				return c, doguRegistryMock, recorder, doguResource
			},
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				repo := &flux.OCIRepository{}
				err := client.Get(t.Context(), types.NamespacedName{Name: testDoguResource.Name, Namespace: testDoguResource.Namespace}, repo)
				assert.NoError(t, err)
				// spec should be unchanged
				assert.Equal(t, testChartURL, repo.Spec.URL)
				assert.Equal(t, "1.2.3-different", repo.Spec.Reference.Tag)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient, doguRegistry, eventRecorder, doguResource := tt.setup(t)
			eor := &EnsureOCIRepositoryStep{k8sClient: k8sClient, doguRegistry: doguRegistry, eventRecorder: eventRecorder}

			if got := eor.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, eor.k8sClient)
			}
		})
	}
}

const testDevChartURL = "oci://k3d-registry-proxy.localhost:5000/testing/dogu/v3/charts/nexus"

func TestEnsureOCIRepositoryStep_Run_devCharts(t *testing.T) {
	t.Run("should use the dev chart from the annotation without asking the registry if dev charts are enabled", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Spec.Version = "1.0.0-dev.1"
		doguResource.Annotations = map[string]string{AnnotationDevChartURL: testDevChartURL}
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, ReasonDevChart, "OCIRepository created from dev chart oci://k3d-registry-proxy.localhost:5000/testing/dogu/v3/charts/nexus:1.0.0-dev.1")

		sut := NewEnsureOCIRepositoryStep(c, NewMockDoguRegistryReader(t), recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.Equal(t, testDevChartURL, repo.Spec.URL)
		assert.Equal(t, "1.0.0-dev.1", repo.Spec.Reference.Tag)
		assert.Equal(t, "1.0.0-dev.1", repo.Labels[v3beta1.DoguLabelVersion])
		assert.False(t, repo.Spec.Insecure)
	})

	t.Run("should mark the repository as insecure if the dev chart is insecure", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Annotations = map[string]string{
			AnnotationDevChartURL:      testDevChartURL,
			AnnotationDevChartInsecure: "true",
		}
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, ReasonDevChart, "OCIRepository created from dev chart oci://k3d-registry-proxy.localhost:5000/testing/dogu/v3/charts/nexus:1.0.0")

		sut := NewEnsureOCIRepositoryStep(c, NewMockDoguRegistryReader(t), recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.True(t, repo.Spec.Insecure)
	})

	t.Run("should not mark the repository as insecure if the dev chart insecure annotation is not true", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Annotations = map[string]string{
			AnnotationDevChartURL:      testDevChartURL,
			AnnotationDevChartInsecure: "false",
		}
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, ReasonDevChart, "OCIRepository created from dev chart oci://k3d-registry-proxy.localhost:5000/testing/dogu/v3/charts/nexus:1.0.0")

		sut := NewEnsureOCIRepositoryStep(c, NewMockDoguRegistryReader(t), recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.False(t, repo.Spec.Insecure)
	})

	t.Run("should update an existing repository to the new dev chart version", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Spec.Version = "1.0.0-dev.2"
		doguResource.Annotations = map[string]string{AnnotationDevChartURL: testDevChartURL}
		existingRepository := testRepository.DeepCopy()
		existingRepository.Spec.URL = testDevChartURL
		existingRepository.Spec.Reference.Tag = "1.0.0-dev.1"
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).Build()

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, ReasonDevChart, "OCIRepository updated from dev chart oci://k3d-registry-proxy.localhost:5000/testing/dogu/v3/charts/nexus:1.0.0-dev.2")

		sut := NewEnsureOCIRepositoryStep(c, NewMockDoguRegistryReader(t), recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.Equal(t, "1.0.0-dev.2", repo.Spec.Reference.Tag)
	})

	t.Run("should ignore the dev chart annotations, warn and use the dogu registry if dev charts are disabled", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Annotations = map[string]string{
			AnnotationDevChartURL:      testDevChartURL,
			AnnotationDevChartInsecure: "true",
		}
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()

		doguRegistryMock := NewMockDoguRegistryReader(t)
		doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeWarning, ReasonDevChartsDisabled, "Ignoring annotation k8s.cloudogu.com/dev-chart-url because dev charts are disabled in the dogu operator; using the dogu registry")
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository created")

		sut := NewEnsureOCIRepositoryStep(c, doguRegistryMock, recorderMock, false)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		checkExpectedRepositoryValues(t, c)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.False(t, repo.Spec.Insecure)
	})

	t.Run("should use the dogu registry without the dev chart annotation if dev charts are enabled", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Annotations = map[string]string{AnnotationDevChartInsecure: "true"}
		existingRepository := testRepository.DeepCopy()
		existingRepository.Spec.URL = testDevChartURL
		existingRepository.Spec.Insecure = true
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).Build()

		doguRegistryMock := NewMockDoguRegistryReader(t)
		doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository updated")

		sut := NewEnsureOCIRepositoryStep(c, doguRegistryMock, recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		checkExpectedRepositoryValues(t, c)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.False(t, repo.Spec.Insecure)
	})

	t.Run("should not apply the dev chart if reconciliation is paused", func(t *testing.T) {
		// given
		doguResource := testDoguResource.DeepCopy()
		doguResource.Spec.PauseReconciliation = true
		doguResource.Annotations = map[string]string{AnnotationDevChartURL: testDevChartURL}
		existingRepository := testRepository.DeepCopy()
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).WithStatusSubresource(doguResource).Build()

		recorderMock := NewMockEventRecorder(t)
		recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionUpdatePending, "Changes not propagated to OCIRepository, because dogu reconciliation is paused")

		sut := NewEnsureOCIRepositoryStep(c, NewMockDoguRegistryReader(t), recorderMock, true)

		// when
		got := sut.Run(testCtx, doguResource)

		// then
		assert.Equal(t, testStepResultContinue, got)
		repo := &flux.OCIRepository{}
		require.NoError(t, c.Get(testCtx, testNamespacedName, repo))
		assert.Equal(t, testChartURL, repo.Spec.URL)
	})
}

func checkExpectedRepositoryValues(t *testing.T, client K8sClient) {
	repo := &flux.OCIRepository{}
	err := client.Get(testCtx, testNamespacedName, repo)

	require.NoError(t, err)
	require.NotNil(t, repo.Labels)
	assert.Equal(t, resource.LabelValueCes, repo.Labels[labelKeyFluxSharding])
	assert.Equal(t, testDoguName, repo.Labels[v3beta1.DoguLabelName])
	assert.Equal(t, testVersion, repo.Labels[v3beta1.DoguLabelVersion])
	assert.Equal(t, testDoguName, repo.Name)
	assert.Equal(t, testNamespace, repo.Namespace)
	assert.Equal(t, testChartURL, repo.Spec.URL)
	assert.Equal(t, testVersion, repo.Spec.Reference.Tag)
	assert.Equal(t, time.Hour*6, repo.Spec.Interval.Duration)
	assert.Equal(t, "ces-container-registries", repo.Spec.SecretRef.Name)
	assert.NotNil(t, repo.OwnerReferences)
	assert.Len(t, repo.OwnerReferences, 1)
	assert.Equal(t, testDoguName, repo.OwnerReferences[0].Name)
	assert.True(t, *repo.OwnerReferences[0].Controller)
	assert.True(t, *repo.OwnerReferences[0].BlockOwnerDeletion)
}

func TestNewEnsureOCIRepositoryStep(t *testing.T) {
	// given
	clientMock := NewMockK8sClient(t)
	doguRegistryMock := NewMockDoguRegistryReader(t)
	recorderMock := NewMockEventRecorder(t)

	// when
	sut := NewEnsureOCIRepositoryStep(clientMock, doguRegistryMock, recorderMock, true)

	// then
	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.k8sClient)
	assert.Equal(t, doguRegistryMock, sut.doguRegistry)
	assert.Equal(t, recorderMock, sut.eventRecorder)
	assert.True(t, sut.devChartsEnabled)
}
