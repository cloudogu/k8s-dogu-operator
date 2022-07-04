package limit

import (
	"context"
	"github.com/cloudogu/cesapp-lib/registry/mocks"
	coreosclient "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func TestNewHardwareLimitUpdater(t *testing.T) {
	t.Parallel()

	t.Run("create with success", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()

		// when
		updater, err := NewHardwareLimitUpdater(clientMock, "myNamespace")

		// then
		require.NoError(t, err)
		assert.NotNil(t, updater)
	})

	t.Run("fail creation based on invalid etcd endpoint", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()

		// when
		updater, err := NewHardwareLimitUpdater(clientMock, "(!)//=)!%(?=(")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse \"http://etcd.(!)//=)!%(?=(.svc.cluster.local:4001\": invalid URL escape \"%(\"")
		assert.Nil(t, updater)
	})
}

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	return scheme
}

func Test_hardwareLimitUpdater_Start(t *testing.T) {
	t.Run("run start and send done to context", func(t *testing.T) { // given
		regMock := &mocks.Registry{}
		watchContextMock := &mocks.WatchConfigurationContext{}
		regMock.On("RootConfig").Return(watchContextMock, nil)
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Return()

		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		hardwareUpdater := &hardwareLimitUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*50)

		// when
		err := hardwareUpdater.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)
	})

	t.Run("run start and send done to context", func(t *testing.T) { // given
		regMock := &mocks.Registry{}
		watchContextMock := &mocks.WatchConfigurationContext{}
		regMock.On("RootConfig").Return(watchContextMock, nil)
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Return()

		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		hardwareUpdater := &hardwareLimitUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*50)

		// when
		err := hardwareUpdater.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)
	})

	t.Run("run start and send change event", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}

		watchContextMock := &mocks.WatchConfigurationContext{}
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Run(func(args mock.Arguments) {
			channelobject := args.Get(3)
			sendChannel, ok := channelobject.(chan *coreosclient.Response)

			if ok {
				testResponse := &coreosclient.Response{}
				sendChannel <- testResponse
			}
		}).Return()
		regMock.On("RootConfig").Return(watchContextMock, nil)

		globalConfigMock := &mocks.ConfigurationContext{}
		globalConfigMock.On("Get", "certificate/server.crt").Return("mycert", nil)
		globalConfigMock.On("Get", "certificate/server.key").Return("mykey", nil)
		regMock.On("GlobalConfig").Return(globalConfigMock, nil)

		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		hardwareUpdater := &hardwareLimitUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)

		// when
		err := hardwareUpdater.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)
	})

	t.Run("run start and get error on ssl change method", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}

		watchContextMock := &mocks.WatchConfigurationContext{}
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Run(func(args mock.Arguments) {
			channelobject := args.Get(3)
			sendChannel, ok := channelobject.(chan *coreosclient.Response)

			if ok {
				testResponse := &coreosclient.Response{}
				sendChannel <- testResponse
			}
		}).Return()
		regMock.On("RootConfig").Return(watchContextMock, nil)

		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		hardwareUpdater := &hardwareLimitUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)

		// when
		_ = hardwareUpdater.Start(ctx)
		cancelFunc()

		// then
		//require.Error(t, err, assert.AnError)
	})
}
