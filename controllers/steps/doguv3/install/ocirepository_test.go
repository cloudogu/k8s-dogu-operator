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
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	flux "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
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
	testChartURL      = "oci://localhost:5000/testing/nexus/1.0.0"
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDoguName,
			Namespace: testNamespace,
		},
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
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				repo := &flux.OCIRepository{}
				err := client.Get(testCtx, testNamespacedName, repo)

				require.NoError(t, err)
				require.NotNil(t, repo.Labels)
				assert.Equal(t, fluxShardingLabelValue, repo.Labels[fluxShardingLabelKey])
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
			},
		},
		{
			name: "should continue on success and update if repository already exists",
			setup: func(t *testing.T) (K8sClient, DoguRegistryReader, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				existingRepository := &flux.OCIRepository{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      testDoguName,
					},
					Spec: flux.OCIRepositorySpec{
						URL: "oci://localhost:5000/testing/nexus/0.9.0",
						Reference: &flux.OCIRepositoryRef{
							Tag: "0.9.0",
						},
					},
				}
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, existingRepository).Build()

				doguRegistryMock := NewMockDoguRegistryReader(t)
				doguRegistryMock.EXPECT().Get(testCtx, testIdentifier).Return(testDoguDescriptor, nil)

				recorderMock := NewMockEventRecorder(t)
				recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository updated")

				return c, doguRegistryMock, recorderMock, doguResource
			},
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				repo := &flux.OCIRepository{}
				err := client.Get(testCtx, testNamespacedName, repo)

				require.NoError(t, err)
				require.NotNil(t, repo.Labels)
				assert.Equal(t, fluxShardingLabelValue, repo.Labels[fluxShardingLabelKey])
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
			},
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
				existingRepository := &flux.OCIRepository{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      testDoguName,
					},
					Spec: flux.OCIRepositorySpec{
						URL: "oci://localhost:5000/testing/nexus/0.9.0",
						Reference: &flux.OCIRepositoryRef{
							Tag: "0.9.0",
						},
					},
				}
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

func TestNewEnsureOCIRepositoryStep(t *testing.T) {
	// given
	clientMock := NewMockK8sClient(t)
	doguRegistryMock := NewMockDoguRegistryReader(t)
	recorderMock := NewMockEventRecorder(t)

	// when
	sut := NewEnsureOCIRepositoryStep(clientMock, doguRegistryMock, recorderMock)

	// then
	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.k8sClient)
	assert.Equal(t, doguRegistryMock, sut.doguRegistry)
	assert.Equal(t, recorderMock, sut.eventRecorder)
}
