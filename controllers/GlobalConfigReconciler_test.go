package controllers

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguclient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestNewGlobalConfigReconciler(t *testing.T) {
	type args struct {
		ecosystemClientFn func(t *testing.T) doguclient.EcoSystemV2Interface
		clientFn          func(t *testing.T) client.Client
		doguEventsFn      func(t *testing.T) chan<- event.TypedGenericEvent[*v2.Dogu]
	}
	tests := []struct {
		name    string
		args    args
		want    GenericReconciler
		wantErr assert.ErrorAssertionFunc
		setupFn func(t *testing.T)
	}{
		{
			name: "should fail to get rest config",
			setupFn: func(t *testing.T) {
				t.Helper()

				oldGetConfig := ctrl.GetConfig
				t.Cleanup(func() {
					ctrl.GetConfig = oldGetConfig
				})
				ctrl.GetConfig = func() (*rest.Config, error) {
					return nil, assert.AnError
				}
			},
			args: args{
				ecosystemClientFn: func(t *testing.T) doguclient.EcoSystemV2Interface {
					m := newMockEcosystemInterface(t)
					return m
				},
				clientFn: func(t *testing.T) client.Client {
					c := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
					return c
				},
				doguEventsFn: func(t *testing.T) chan<- event.TypedGenericEvent[*v2.Dogu] {
					ch := make(chan event.TypedGenericEvent[*v2.Dogu])
					return ch
				},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get rest config")
			},
		},
		{
			name: "should fail to get client set",
			setupFn: func(t *testing.T) {
				t.Helper()

				oldGetConfig := ctrl.GetConfig
				t.Cleanup(func() {
					ctrl.GetConfig = oldGetConfig
				})
				ctrl.GetConfig = func() (*rest.Config, error) {
					return nil, assert.AnError
				}

				oldClientSetGetter := clientSetGetter
				t.Cleanup(func() {
					clientSetGetter = oldClientSetGetter
				})
				clientSetGetter = func(c *rest.Config) (kubernetes.Interface, error) {
					return nil, assert.AnError
				}
			},
			args: args{
				ecosystemClientFn: func(t *testing.T) doguclient.EcoSystemV2Interface {
					m := newMockEcosystemInterface(t)
					return m
				},
				clientFn: func(t *testing.T) client.Client {
					c := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
					return c
				},
				doguEventsFn: func(t *testing.T) chan<- event.TypedGenericEvent[*v2.Dogu] {
					ch := make(chan event.TypedGenericEvent[*v2.Dogu])
					return ch
				},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get rest config")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGlobalConfigReconciler(tt.args.ecosystemClientFn(t), tt.args.clientFn(t), util.testNamespace, tt.args.doguEventsFn(t))
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
