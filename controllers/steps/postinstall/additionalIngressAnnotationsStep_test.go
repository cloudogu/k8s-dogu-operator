package postinstall

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v3 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var testCtx = context.Background()

const namespace = "ecosystem"

func TestNewAdditionalIngressAnnotationsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewAdditionalIngressAnnotationsStep(newMockK8sClient(t))

		assert.NotNil(t, step)
	})
}

func TestAdditionalIngressAnnotationsStep_Run(t *testing.T) {
	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get service",
			clientFn: func(t *testing.T) k8sClient {
				mck := newMockK8sClient(t)
				mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v1.Service{}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v3.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should not be change",
			clientFn: func(t *testing.T) k8sClient {
				mck := newMockK8sClient(t)
				mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v1.Service{}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v3.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aias := &AdditionalIngressAnnotationsStep{
				client: tt.clientFn(t),
			}
			assert.Equalf(t, tt.want, aias.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func TestAdditionalIngressAnnotationsStep_checkForAdditionalIngressAnnotations(t *testing.T) {
	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		doguService  *v1.Service
		doguResource *v2.Dogu
		want         bool
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to unmarshal json annotation",
			clientFn: func(t *testing.T) k8sClient {
				return newMockK8sClient(t)
			},
			doguResource: &v2.Dogu{},
			doguService: &v1.Service{
				ObjectMeta: v3.ObjectMeta{
					Annotations: map[string]string{
						annotation.AdditionalIngressAnnotationsAnnotation: "{]",
					},
				},
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to get additional ingress annotations from service of dogu")
				return true
			},
		},
		{
			name: "Annotations of doguService and resource should be different",
			clientFn: func(t *testing.T) k8sClient {
				return newMockK8sClient(t)
			},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					AdditionalIngressAnnotations: map[string]string{
						"test": "test",
					},
				},
			},
			doguService: &v1.Service{
				ObjectMeta: v3.ObjectMeta{
					Annotations: map[string]string{
						annotation.AdditionalIngressAnnotationsAnnotation: "{}",
					},
				},
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "Annotations of doguService and resource should be the same",
			clientFn: func(t *testing.T) k8sClient {
				return newMockK8sClient(t)
			},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					AdditionalIngressAnnotations: map[string]string{},
				},
			},
			doguService: &v1.Service{
				ObjectMeta: v3.ObjectMeta{
					Annotations: map[string]string{
						annotation.AdditionalIngressAnnotationsAnnotation: "{}",
					},
				},
			},
			want:    false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aias := &AdditionalIngressAnnotationsStep{
				client: tt.clientFn(t),
			}
			got, err := aias.checkForAdditionalIngressAnnotations(tt.doguService, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("checkForAdditionalIngressAnnotations(%v, %v)", tt.doguService, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "checkForAdditionalIngressAnnotations(%v, %v)", tt.doguService, tt.doguResource)
		})
	}
}

func TestAdditionalIngressAnnotationsStep_setDoguAdditionalIngressAnnotations(t *testing.T) {
	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		annotatorFn  func(t *testing.T) ingressAnnotator
		doguService  *v1.Service
		doguResource *v2.Dogu
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "Should fail to add additional ingress annotation to service",
			clientFn: func(t *testing.T) k8sClient {
				return newMockK8sClient(t)
			},
			annotatorFn: func(t *testing.T) ingressAnnotator {
				mck := newMockIngressAnnotator(t)
				mck.EXPECT().AppendIngressAnnotationsToService(&v1.Service{}, v2.IngressAnnotations{}).Return(assert.AnError)
				return mck
			},
			doguService: &v1.Service{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					AdditionalIngressAnnotations: v2.IngressAnnotations{},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to add additional ingress annotations to service of dogu")
				return true
			},
		},
		{
			name: "Should fail to update service",
			clientFn: func(t *testing.T) k8sClient {
				mck := newMockK8sClient(t)
				mck.EXPECT().Update(testCtx, &v1.Service{}).Return(assert.AnError)
				return mck
			},
			annotatorFn: func(t *testing.T) ingressAnnotator {
				mck := newMockIngressAnnotator(t)
				mck.EXPECT().AppendIngressAnnotationsToService(&v1.Service{}, v2.IngressAnnotations{}).Return(nil)
				return mck
			},
			doguService: &v1.Service{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					AdditionalIngressAnnotations: v2.IngressAnnotations{},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to update dogu service ")
				return true
			},
		},
		{
			name: "Should succeed to update dogu service",
			clientFn: func(t *testing.T) k8sClient {
				mck := newMockK8sClient(t)
				mck.EXPECT().Update(testCtx, &v1.Service{}).Return(nil)
				return mck
			},
			annotatorFn: func(t *testing.T) ingressAnnotator {
				mck := newMockIngressAnnotator(t)
				mck.EXPECT().AppendIngressAnnotationsToService(&v1.Service{}, v2.IngressAnnotations{}).Return(nil)
				return mck
			},
			doguService: &v1.Service{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					AdditionalIngressAnnotations: v2.IngressAnnotations{},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aias := &AdditionalIngressAnnotationsStep{
				client:    tt.clientFn(t),
				annotator: tt.annotatorFn(t),
			}
			tt.wantErr(t, aias.setDoguAdditionalIngressAnnotations(testCtx, tt.doguService, tt.doguResource), fmt.Sprintf("setDoguAdditionalIngressAnnotations(%v, %v, %v)", testCtx, tt.doguService, tt.doguResource))
		})
	}
}
