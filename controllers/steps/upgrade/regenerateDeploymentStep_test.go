package upgrade

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeploymentUpdaterStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		deployInterface := newMockDeploymentInterface(t)
		resourceGen := newMockResourceGenerator(t)
		step := NewRegenerateDeploymentStep(
			fetcher,
			deployInterface,
			resourceGen,
		)

		assert.NotNil(t, step)
		assert.Equal(t, fetcher, step.localDoguFetcher)
		assert.Equal(t, deployInterface, step.deploymentInterface)
		assert.Equal(t, resourceGen, step.resourceGenerator)
	})
}

func TestDeploymentUpdaterStep_Run(t *testing.T) {
	doguUpgradeResource := &v2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v2.DoguSpec{
			Version: "1.0.1",
		},
		Status: v2.DoguStatus{
			InstalledVersion: "1.0.0",
		},
	}

	doguNoUpgradeResource := &v2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v2.DoguSpec{
			Version: "1.0.0",
		},
		Status: v2.DoguStatus{
			InstalledVersion: "1.0.0",
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	deploymentToUpdate := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Labels: map[string]string{"foo": "bar"}},
	}

	doguDescriptor := &core.Dogu{Name: "test"}

	type fields struct {
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		resourceGeneratorFn   func(t *testing.T) resourceGenerator
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should do nothing on upgrade (deployment should already be updated earlier)",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					return newMockDeploymentInterface(t)
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					return newMockResourceGenerator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
			},
			doguResource: doguUpgradeResource,
			want:         steps.Continue(),
		},
		{
			name: "should requeue on deployment fetch error",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, doguNoUpgradeResource.Name, metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					return newMockResourceGenerator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should requeue on dogu fetch error",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, doguNoUpgradeResource.Name, metav1.GetOptions{}).Return(deployment, nil)
					return mck
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					return newMockResourceGenerator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to create update deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, doguNoUpgradeResource.Name, metav1.GetOptions{}).Return(deployment, nil)
					return mck
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					mck := newMockResourceGenerator(t)
					mck.EXPECT().UpdateDoguDeployment(testCtx, deployment, doguNoUpgradeResource, doguDescriptor).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(doguDescriptor, nil)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, doguNoUpgradeResource.Name, metav1.GetOptions{}).Return(deployment, nil)
					mck.EXPECT().Update(testCtx, deploymentToUpdate, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					mck := newMockResourceGenerator(t)
					mck.EXPECT().UpdateDoguDeployment(testCtx, deployment, doguNoUpgradeResource, doguDescriptor).Return(deploymentToUpdate, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(doguDescriptor, nil)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, doguNoUpgradeResource.Name, metav1.GetOptions{}).Return(deployment, nil)
					mck.EXPECT().Update(testCtx, deploymentToUpdate, metav1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
				resourceGeneratorFn: func(t *testing.T) resourceGenerator {
					mck := newMockResourceGenerator(t)
					mck.EXPECT().UpdateDoguDeployment(testCtx, deployment, doguNoUpgradeResource, doguDescriptor).Return(deploymentToUpdate, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(doguDescriptor, nil)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dus := &RegenerateDeploymentStep{
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				resourceGenerator:   tt.fields.resourceGeneratorFn(t),
			}
			assert.Equalf(t, tt.want, dus.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
