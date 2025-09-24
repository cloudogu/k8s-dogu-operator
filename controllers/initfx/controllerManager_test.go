package initfx

import (
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func Test_noSpamKey(t *testing.T) {
	t.Run("should not return the same key", func(t *testing.T) {
		// when
		first := noSpamKey(nil)
		second := noSpamKey(nil)

		// then
		assert.NotEqual(t, first, second)
	})
}

func Test_noAggregationKey(t *testing.T) {
	t.Run("should not return the same keys", func(t *testing.T) {
		// when
		firstAggregateKey, firstLocalKey := noAggregationKey(nil)
		secondAggregateKey, secondLocalKey := noAggregationKey(nil)

		// then
		assert.NotEqual(t, firstAggregateKey, secondAggregateKey)
		assert.NotEqual(t, firstLocalKey, secondLocalKey)
	})
}

func Test_addChecks(t *testing.T) {
	t.Run("should fail to add health check", func(t *testing.T) {
		// given
		managerMock := newMockK8sManager(t)
		managerMock.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(assert.AnError)

		// when
		err := addChecks(managerMock)

		// then
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to add healthz check:")
	})
	t.Run("should fail to add ready check", func(t *testing.T) {
		// given
		managerMock := newMockK8sManager(t)
		managerMock.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(nil)
		managerMock.EXPECT().AddReadyzCheck("readyz", mock.AnythingOfType("healthz.Checker")).Return(assert.AnError)

		// when
		err := addChecks(managerMock)

		// then
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to add readyz check:")
	})
	t.Run("should fail to add ready check", func(t *testing.T) {
		// given
		managerMock := newMockK8sManager(t)
		managerMock.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(nil)
		managerMock.EXPECT().AddReadyzCheck("readyz", mock.AnythingOfType("healthz.Checker")).Return(nil)

		// when
		err := addChecks(managerMock)

		// then
		assert.NoError(t, err)
	})
}

func TestNewManagerOptions(t *testing.T) {
	t.Run("should create manager options", func(t *testing.T) {
		// given
		operatorConfig := &config.OperatorConfig{}

		// when
		options := NewManagerOptions(operatorConfig)

		// then
		assert.Equal(t, server.Options{BindAddress: ":8080"}, options.Metrics)
		assert.Equal(t, cache.Options{DefaultNamespaces: map[string]cache.Config{
			operatorConfig.Namespace: {},
		}}, options.Cache)
		assert.Equal(t, webhook.NewServer(webhook.Options{Port: 9443}), options.WebhookServer)
		assert.Equal(t, ":8081", options.HealthProbeBindAddress)
		assert.Equal(t, false, options.LeaderElection)
		assert.Equal(t, "951e217a.cloudogu.com", options.LeaderElectionID)
	})
}

func TestNewControllerManager(t *testing.T) {
	type args struct {
		lcFn         func(t *testing.T) fxLifecycle
		loggerFn     func(t *testing.T) logr.Logger
		optionsFn    func(t *testing.T) manager.Options
		restConfigFn func(t *testing.T) *rest.Config
	}
	tests := []struct {
		name    string
		args    args
		want    manager.Manager
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to create manager",
			args: args{
				restConfigFn: func(t *testing.T) *rest.Config {
					return nil
				},
				lcFn: func(t *testing.T) fxLifecycle {
					return newMockFxLifecycle(t)
				},
				loggerFn: func(t *testing.T) logr.Logger {
					return logr.Logger{}
				},
				optionsFn: func(t *testing.T) manager.Options {
					return NewManagerOptions(&config.OperatorConfig{})
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewControllerManager(tt.args.lcFn(t), tt.args.loggerFn(t), tt.args.optionsFn(t), tt.args.restConfigFn(t))
			if !tt.wantErr(t, err, fmt.Sprintf("NewControllerManager(%v, %v, %v, %v)", tt.args.lcFn(t), tt.args.loggerFn(t), tt.args.optionsFn(t), tt.args.restConfigFn(t))) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewControllerManager(%v, %v, %v, %v)", tt.args.lcFn(t), tt.args.loggerFn(t), tt.args.optionsFn(t), tt.args.restConfigFn(t))
		})
	}
}
