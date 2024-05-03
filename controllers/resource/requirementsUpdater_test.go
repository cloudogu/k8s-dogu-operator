package resource

import (
	"context"
	"errors"
	"github.com/cloudogu/cesapp-lib/core"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

	coreosclient "go.etcd.io/etcd/client/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewRequirementsUpdater(t *testing.T) {
	t.Parallel()

	t.Run("create with success", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		configMapClient := extMocks.NewConfigMapInterface(t)
		coreV1Client := extMocks.NewCoreV1Interface(t)
		coreV1Client.EXPECT().ConfigMaps("myNamespace").Return(configMapClient)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		// when
		updater, err := NewRequirementsUpdater(clientMock, "myNamespace", clientSetMock)

		// then
		require.NoError(t, err)
		assert.NotNil(t, updater)
	})

	t.Run("fail creation based on invalid etcd endpoint", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		clientSetMock := extMocks.NewClientSet(t)

		// when
		updater, err := NewRequirementsUpdater(clientMock, "(!)//=)!%(?=(", clientSetMock)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "parse \"http://etcd.(!)//=)!%(?=(.svc.cluster.local:4001\": invalid URL escape \"%(\"")
		assert.Nil(t, updater)
	})
}

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))
	return scheme
}

func getTestDoguJsons() (*core.Dogu, *core.Dogu, *core.Dogu) {
	dogu1 := &core.Dogu{Name: "official/dogu1"}
	dogu2 := &core.Dogu{Name: "official/dogu2"}
	dogu3 := &core.Dogu{Name: "official/dogu3"}
	return dogu1, dogu2, dogu3
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
		Spec: appsv1.DeploymentSpec{Template: v1.PodTemplateSpec{Spec: v1.PodSpec{
			Containers: []v1.Container{
				{},
			},
		}}},
	}
	dogu2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu2",
			Namespace: "test",
		},
		Spec: appsv1.DeploymentSpec{Template: v1.PodTemplateSpec{Spec: v1.PodSpec{
			Containers: []v1.Container{
				{},
			},
		}}},
	}
	dogu3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dogu3",
			Namespace: "test",
		},
		Spec: appsv1.DeploymentSpec{Template: v1.PodTemplateSpec{Spec: v1.PodSpec{
			Containers: []v1.Container{
				{},
			},
		}}},
	}
	return dogu1, dogu2, dogu3
}

func Test_requirementsUpdater_Start(t *testing.T) {
	t.Run("run start and send done to context", func(t *testing.T) { // given
		regMock := &cesmocks.Registry{}
		watchContextMock := &cesmocks.WatchConfigurationContext{}
		regMock.On("RootConfig").Return(watchContextMock, nil)
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Return()

		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		sut := &requirementsUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(testCtx, time.Millisecond*50)

		// when
		err := sut.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)
	})

	t.Run("run start and send change event", func(t *testing.T) {
		// given
		regMock := extMocks.NewConfigurationRegistry(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu1").Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu2").Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu3").Return(dj3, nil)

		watchContextMock := &cesmocks.WatchConfigurationContext{}
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Run(func(args mock.Arguments) {
			channelobject := args.Get(3)
			sendChannel, ok := channelobject.(chan *coreosclient.Response)

			if ok {
				testResponse := &coreosclient.Response{}
				sendChannel <- testResponse
			}
		}).Return()
		regMock.EXPECT().RootConfig().Return(watchContextMock)

		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		generator := mocks.NewResourceRequirementsGenerator(t)
		generator.EXPECT().Generate(dj1).Return(v1.ResourceRequirements{Limits: v1.ResourceList{v1.ResourceMemory: resource.MustParse("500Mi")}}, nil)
		generator.EXPECT().Generate(dj2).Return(v1.ResourceRequirements{Requests: v1.ResourceList{v1.ResourceEphemeralStorage: resource.MustParse("2Gi")}}, nil)
		generator.EXPECT().Generate(dj3).Return(v1.ResourceRequirements{Limits: v1.ResourceList{v1.ResourceCPU: resource.MustParse("500m")}}, nil)

		sut := &requirementsUpdater{
			client:            clientMock,
			registry:          regMock,
			requirementsGen:   generator,
			localDoguRegistry: localDoguRegMock,
		}

		ctx, cancelFunc := context.WithTimeout(testCtx, time.Second*2)

		// when
		err := sut.Start(ctx)
		cancelFunc()

		// then
		require.NoError(t, err)

		doguDeployment1 := &appsv1.Deployment{}
		err = clientMock.Get(ctx, types.NamespacedName{Name: dd1.GetName(), Namespace: dd1.GetNamespace()}, doguDeployment1)
		assert.NoError(t, err)
		assert.Equal(t, resource.MustParse("500Mi"), doguDeployment1.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceMemory])
		doguDeployment2 := &appsv1.Deployment{}
		err = clientMock.Get(ctx, types.NamespacedName{Name: dd2.GetName(), Namespace: dd2.GetNamespace()}, doguDeployment2)
		assert.NoError(t, err)
		assert.Equal(t, resource.MustParse("2Gi"), doguDeployment2.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceEphemeralStorage])
		doguDeployment3 := &appsv1.Deployment{}
		err = clientMock.Get(ctx, types.NamespacedName{Name: dd3.GetName(), Namespace: dd3.GetNamespace()}, doguDeployment3)
		assert.NoError(t, err)
		assert.Equal(t, resource.MustParse("500m"), doguDeployment3.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceCPU])
	})

	t.Run("run start and get error on etcd change method", func(t *testing.T) {
		// given
		regMock := extMocks.NewConfigurationRegistry(t)

		watchContextMock := &cesmocks.WatchConfigurationContext{}
		watchContextMock.On("Watch", mock.Anything, triggerSyncEtcdKeyFullPath, false, mock.Anything).Run(func(args mock.Arguments) {
			channelobject := args.Get(3)
			sendChannel, ok := channelobject.(chan *coreosclient.Response)

			if ok {
				testResponse := &coreosclient.Response{}
				sendChannel <- testResponse
			}
		}).Return()
		regMock.EXPECT().RootConfig().Return(watchContextMock)

		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			}}).
			Build()
		sut := &requirementsUpdater{
			client:   clientMock,
			registry: regMock,
		}

		ctx, cancelFunc := context.WithTimeout(testCtx, time.Second*2)

		// when
		err := sut.Start(ctx)
		cancelFunc()

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}

func Test_requirementsUpdater_triggerSync(t *testing.T) {
	t.Run("trigger fail on retrieving dogus", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().
			Build()

		sut := &requirementsUpdater{
			client: clientMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get installed dogus from the cluster: failed to list dogus in namespace")
	})

	t.Run("trigger fail on retrieving dogu.jsons", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			Build()

		generator := mocks.NewResourceRequirementsGenerator(t)
		regMock := extMocks.NewConfigurationRegistry(t)
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d1.Name).Return(nil, assert.AnError)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d2.Name).Return(nil, assert.AnError)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d3.Name).Return(nil, assert.AnError)

		sut := &requirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			registry:          regMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		assert.ErrorContains(t, err, "failed to get dogu.json of dogu [dogu1] from registry")
		assert.ErrorContains(t, err, "failed to get dogu.json of dogu [dogu2] from registry")
		assert.ErrorContains(t, err, "failed to get dogu.json of dogu [dogu3] from registry")
	})

	t.Run("trigger fail on retrieving dogu deployments", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			Build()

		generator := mocks.NewResourceRequirementsGenerator(t)
		regMock := extMocks.NewConfigurationRegistry(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &requirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			registry:          regMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		assert.ErrorContains(t, err, "failed to get deployment of dogu [test/dogu1]")
		assert.ErrorContains(t, err, "failed to get deployment of dogu [test/dogu2]")
		assert.ErrorContains(t, err, "failed to get deployment of dogu [test/dogu3]")
	})

	t.Run("trigger fail on generating resource requirements", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		generator := mocks.NewResourceRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		testErr1 := errors.New("error1 occurred: wrong bitsize")
		testErr2 := errors.New("error2 occurred: out of entropy")
		testErr3 := errors.New("error3 failed to fail: bad luck")
		generator.EXPECT().Generate(dj1).Return(v1.ResourceRequirements{}, testErr1)
		generator.EXPECT().Generate(dj2).Return(v1.ResourceRequirements{}, testErr2)
		generator.EXPECT().Generate(dj3).Return(v1.ResourceRequirements{}, testErr3)

		regMock := extMocks.NewConfigurationRegistry(t)
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &requirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			registry:          regMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to generate resource requirements of dogu")
		assert.ErrorContains(t, err, "test/dogu1")
		assert.ErrorContains(t, err, "test/dogu2")
		assert.ErrorContains(t, err, "test/dogu3")
		assert.ErrorIs(t, err, testErr1)
		assert.ErrorIs(t, err, testErr2)
		assert.ErrorIs(t, err, testErr3)
	})

	t.Run("trigger fail on updating deployment", func(t *testing.T) {
		// given
		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			WithInterceptorFuncs(interceptor.Funcs{Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return assert.AnError
			}}).
			Build()

		generator := mocks.NewResourceRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		generator.EXPECT().Generate(dj1).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(dj2).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(dj3).Return(v1.ResourceRequirements{}, nil)

		regMock := extMocks.NewConfigurationRegistry(t)
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &requirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			registry:          regMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "test/dogu1")
		assert.ErrorContains(t, err, "test/dogu2")
		assert.ErrorContains(t, err, "test/dogu3")
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

		generator := mocks.NewResourceRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		generator.EXPECT().Generate(dj1).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(dj2).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(dj3).Return(v1.ResourceRequirements{}, nil)

		regMock := extMocks.NewConfigurationRegistry(t)
		localDoguRegMock := mocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &requirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			registry:          regMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		require.NoError(t, err)
	})
}
