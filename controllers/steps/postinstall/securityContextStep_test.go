package postinstall

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v4 "k8s.io/api/apps/v1"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSecurityContextStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		generator := newMockSecurityContextGenerator(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)

		step := NewSecurityContextStep(fetcher, generator, deploymentInterfaceMock)

		assert.NotNil(t, step)
	})
}

func TestSecurityContextStep_Run(t *testing.T) {
	type fields struct {
		localDoguFetcherFn         func(t *testing.T) localDoguFetcher
		securityContextGeneratorFn func(t *testing.T) securityContextGenerator
		deploymentInterfaceFn      func(t *testing.T) deploymentInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, assert.AnError)
					return mck
				},
				securityContextGeneratorFn: func(t *testing.T) securityContextGenerator {
					return newMockSecurityContextGenerator(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					return newMockDeploymentInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get local descriptor for dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to get deployment",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				securityContextGeneratorFn: func(t *testing.T) securityContextGenerator {
					return newMockSecurityContextGenerator(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to update deployment",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				securityContextGeneratorFn: func(t *testing.T) securityContextGenerator {
					mck := newMockSecurityContextGenerator(t)
					mck.EXPECT().Generate(testCtx, &core.Dogu{}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}).Return(&v3.PodSecurityContext{}, &v3.SecurityContext{})
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v4.Deployment{}, nil)
					mck.EXPECT().Update(testCtx, &v4.Deployment{
						Spec: v4.DeploymentSpec{
							Template: v3.PodTemplateSpec{
								Spec: v3.PodSpec{
									SecurityContext: &v3.PodSecurityContext{},
								},
							},
						},
					}, v1.UpdateOptions{}).Return(&v4.Deployment{}, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update deployment",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				securityContextGeneratorFn: func(t *testing.T) securityContextGenerator {
					mck := newMockSecurityContextGenerator(t)
					mck.EXPECT().Generate(testCtx, &core.Dogu{}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}).Return(&v3.PodSecurityContext{}, &v3.SecurityContext{})
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v4.Deployment{}, nil)
					mck.EXPECT().Update(testCtx, &v4.Deployment{
						Spec: v4.DeploymentSpec{
							Template: v3.PodTemplateSpec{
								Spec: v3.PodSpec{
									SecurityContext: &v3.PodSecurityContext{},
								},
							},
						},
					}, v1.UpdateOptions{}).Return(&v4.Deployment{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scs := &SecurityContextStep{
				localDoguFetcher:         tt.fields.localDoguFetcherFn(t),
				securityContextGenerator: tt.fields.securityContextGeneratorFn(t),
				deploymentInterface:      tt.fields.deploymentInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, scs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
