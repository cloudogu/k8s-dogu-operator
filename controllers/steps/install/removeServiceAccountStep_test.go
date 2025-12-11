package install

import (
	"context"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRemoveServiceAccountStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRemoveServiceAccountStep(newMockServiceAccountRemover(t), newMockLocalDoguFetcher(t), newMockDoguInterface(t))

		assert.NotNil(t, step)
	})
}

func TestRemoveServiceAccountStep_Run(t *testing.T) {
	testCtx := context.TODO()

	type fields struct {
		serviceAccountRemoverFn func(t *testing.T) serviceAccountRemover
		localDoguFetcherFn      func(t *testing.T) localDoguFetcher
		doguInterfaceFn         func(t *testing.T) doguInterface
	}
	doguName := dogu.SimpleName("test")
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch installed dogu",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					return newMockServiceAccountRemover(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, doguName).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"wasRestored": "true"},
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should continue if no annotations are present",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					return newMockServiceAccountRemover(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should continue if restored annotation is missing",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					return newMockServiceAccountRemover(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"other": "value"},
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to remove service accounts",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					mck := newMockServiceAccountRemover(t)
					mck.EXPECT().RemoveAllFromComponents(testCtx, &cesappcore.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, doguName).Return(&cesappcore.Dogu{Name: "test"}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"wasRestored": "true"},
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update dogu resource",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					mck := newMockServiceAccountRemover(t)
					mck.EXPECT().RemoveAllFromComponents(testCtx, &cesappcore.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, doguName).Return(&cesappcore.Dogu{Name: "test"}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					// The annotation is removed before update
					expectedDogu := &v2.Dogu{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test",
							Annotations: map[string]string{},
						},
					}
					mck.EXPECT().Update(testCtx, expectedDogu, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"wasRestored": "true"},
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed removing service accounts and updating dogu",
			fields: fields{
				serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
					mck := newMockServiceAccountRemover(t)
					mck.EXPECT().RemoveAllFromComponents(testCtx, &cesappcore.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, doguName).Return(&cesappcore.Dogu{Name: "test"}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					expectedDogu := &v2.Dogu{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test",
							Annotations: map[string]string{},
						},
					}
					mck.EXPECT().Update(testCtx, expectedDogu, metav1.UpdateOptions{}).Return(expectedDogu, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"wasRestored": "true"},
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &RemoveServiceAccountStep{
				serviceAccountRemover: tt.fields.serviceAccountRemoverFn(t),
				localDoguFetcher:      tt.fields.localDoguFetcherFn(t),
				doguInterface:         tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, step.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
