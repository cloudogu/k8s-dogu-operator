package limit

import (
	"context"
	"testing"
	"time"

	coreosclient "github.com/coreos/etcd/client"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
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
	utilruntime.Must(k8sv1.AddToScheme(scheme))
	return scheme
}

func getTestDogus() (*k8sv1.Dogu, *k8sv1.Dogu, *k8sv1.Dogu) {
	dogu1 := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu1",
			Namespace: "test",
		},
	}
	dogu2 := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu2",
			Namespace: "test",
		},
	}
	dogu3 := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu3",
			Namespace: "test",
		},
	}
	return dogu1, dogu2, dogu3
}

func getTestDeployments() (*appsv1.Deployment, *appsv1.Deployment, *appsv1.Deployment) {
	dogu1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu1",
			Namespace: "test",
		},
	}
	dogu2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu2",
			Namespace: "test",
		},
	}
	dogu3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu3",
			Namespace: "test",
		},
	}
	return dogu1, dogu2, dogu3
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

		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		limitPatcher := newLimitPatcher(t)
		limitPatcher.On("RetrievePodLimits", mock.Anything).Return(DoguLimits{}, nil)
		limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)

		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			registry:         regMock,
			doguLimitPatcher: limitPatcher,
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)

		// when
		err := hardwareUpdater.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)
	})

	t.Run("run start and get error on etcd change method", func(t *testing.T) {
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
	})
}

func Test_hardwareLimitUpdater_triggerSync(t *testing.T) {
	t.Run("trigger fail on retrieving dogus", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().
			Build()

		limitPatcher := newLimitPatcher(t)
		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			doguLimitPatcher: limitPatcher,
		}

		// when
		err := hardwareUpdater.triggerSync(context.Background())

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get installed dogus from the cluster: failed to list dogus in namespace")
	})

	t.Run("trigger fail on retrieving dogu deployments", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			Build()

		limitPatcher := newLimitPatcher(t)
		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			doguLimitPatcher: limitPatcher,
		}

		// when
		err := hardwareUpdater.triggerSync(context.Background())

		// then
		var myMultiError *multierror.Error
		require.ErrorAs(t, err, &myMultiError)

		assert.Len(t, myMultiError.Errors, 3)
		assert.Contains(t, myMultiError.Errors[0].Error(), "failed to get deployment of dogu [test/dogu1]")
		assert.Contains(t, myMultiError.Errors[1].Error(), "failed to get deployment of dogu [test/dogu2]")
		assert.Contains(t, myMultiError.Errors[2].Error(), "failed to get deployment of dogu [test/dogu3]")
	})

	t.Run("trigger fail on retrieving memory limits", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		limitPatcher := newLimitPatcher(t)
		limitPatcher.On("RetrievePodLimits", mock.Anything).Return(DoguLimits{}, assert.AnError)
		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			doguLimitPatcher: limitPatcher,
		}

		// when
		err := hardwareUpdater.triggerSync(context.Background())

		// then
		var myMultiError *multierror.Error
		require.ErrorAs(t, err, &myMultiError)

		assert.Len(t, myMultiError.Errors, 3)
		assert.ErrorIs(t, myMultiError.Errors[0], assert.AnError)
		assert.ErrorIs(t, myMultiError.Errors[1], assert.AnError)
		assert.ErrorIs(t, myMultiError.Errors[2], assert.AnError)
	})

	t.Run("trigger fail on patching deployment", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		limitPatcher := newLimitPatcher(t)
		limitPatcher.On("RetrievePodLimits", mock.Anything).Return(DoguLimits{}, nil)
		limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(assert.AnError)
		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			doguLimitPatcher: limitPatcher,
		}

		// when
		err := hardwareUpdater.triggerSync(context.Background())

		// then
		var myMultiError *multierror.Error
		require.ErrorAs(t, err, &myMultiError)

		assert.Len(t, myMultiError.Errors, 3)
		assert.ErrorIs(t, myMultiError.Errors[0], assert.AnError)
		assert.ErrorIs(t, myMultiError.Errors[1], assert.AnError)
		assert.ErrorIs(t, myMultiError.Errors[2], assert.AnError)
	})

	t.Run("trigger success", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		limitPatcher := newLimitPatcher(t)
		limitPatcher.On("RetrievePodLimits", mock.Anything).Return(DoguLimits{}, nil)
		limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)

		hardwareUpdater := &hardwareLimitUpdater{
			client:           clientMock,
			doguLimitPatcher: limitPatcher,
		}

		// when
		err := hardwareUpdater.triggerSync(context.Background())

		// then
		require.NoError(t, err)
	})
}
