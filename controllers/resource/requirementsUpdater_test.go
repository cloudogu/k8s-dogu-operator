package resource

import (
	"context"
	"errors"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sync"
	"testing"
)

func TestNewRequirementsUpdater(t *testing.T) {
	t.Run("create with success", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().WithScheme(getScheme()).Build()
		doguRepoMock := newMockDoguConfigGetter(t)
		doguRegMock := newMockDoguGetter(t)
		watcherMock := newMockGlobalConfigurationWatcher(t)

		// when
		updater, err := NewRequirementsUpdater(clientMock, "myNamespace", doguRepoMock, doguRegMock, watcherMock)

		// then
		require.NoError(t, err)
		assert.NotNil(t, updater)
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
		resultChan := make(chan repository.GlobalConfigWatchResult)

		globalConfigWatcherMock := newMockGlobalConfigurationWatcher(t)
		globalConfigWatcherMock.EXPECT().Watch(mock.Anything, mock.Anything).Return(resultChan, nil)

		sut := &RequirementsUpdater{
			globalConfigWatcher: globalConfigWatcherMock,
		}

		ctx, cancelFunc := context.WithCancel(context.TODO())

		var wg sync.WaitGroup

		// when
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := sut.Start(ctx)
			assert.NoError(t, err)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			<-ctx.Done()
			close(resultChan)
		}()

		cancelFunc()

		wg.Wait()
	})

	t.Run("run start and send change event", func(t *testing.T) {
		// given
		dj1, dj2, dj3 := getTestDoguJsons()
		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu1").Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu2").Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, "dogu3").Return(dj3, nil)

		resultChan := make(chan repository.GlobalConfigWatchResult)

		globalConfigWatcher := newMockGlobalConfigurationWatcher(t)
		globalConfigWatcher.EXPECT().Watch(mock.Anything, mock.Anything).Return(resultChan, nil)

		d1, d2, d3 := getTestDogus()
		dd1, dd2, dd3 := getTestDeployments()
		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithObjects(d1, d2, d3).
			WithObjects(dd1, dd2, dd3).
			Build()

		generator := newMockRequirementsGenerator(t)
		generator.EXPECT().Generate(mock.Anything, dj1).Return(v1.ResourceRequirements{Limits: v1.ResourceList{v1.ResourceMemory: resource.MustParse("500Mi")}}, nil)
		generator.EXPECT().Generate(mock.Anything, dj2).Return(v1.ResourceRequirements{Requests: v1.ResourceList{v1.ResourceEphemeralStorage: resource.MustParse("2Gi")}}, nil)
		generator.EXPECT().Generate(mock.Anything, dj3).Return(v1.ResourceRequirements{Limits: v1.ResourceList{v1.ResourceCPU: resource.MustParse("500m")}}, nil)

		sut := &RequirementsUpdater{
			client:              clientMock,
			requirementsGen:     generator,
			localDoguRegistry:   localDoguRegMock,
			globalConfigWatcher: globalConfigWatcher,
		}

		ctx, cancelFunc := context.WithCancel(testCtx)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			lErr := sut.Start(ctx)
			assert.NoError(t, lErr)
		}()

		wg.Add(1)
		go func() {
			defer close(resultChan)
			defer wg.Done()

			<-ctx.Done()
		}()

		resultChan <- repository.GlobalConfigWatchResult{
			PrevState: config.GlobalConfig{},
			NewState:  config.GlobalConfig{},
			Err:       nil,
		}

		// when
		cancelFunc()

		wg.Wait()

		// then
		doguDeployment1 := &appsv1.Deployment{}
		err := clientMock.Get(ctx, types.NamespacedName{Name: dd1.GetName(), Namespace: dd1.GetNamespace()}, doguDeployment1)
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

	t.Run("run start and get error on change method", func(t *testing.T) {
		// given
		resultChan := make(chan repository.GlobalConfigWatchResult)

		globalConfigWatcher := newMockGlobalConfigurationWatcher(t)
		globalConfigWatcher.EXPECT().Watch(mock.Anything, mock.Anything).Return(resultChan, nil)

		clientMock := testclient.NewClientBuilder().
			WithScheme(getScheme()).
			WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			}}).
			Build()
		sut := &RequirementsUpdater{
			client:              clientMock,
			globalConfigWatcher: globalConfigWatcher,
		}

		ctx, cancelFunc := context.WithCancel(testCtx)

		// when
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			err := sut.Start(ctx)
			assert.ErrorIs(t, err, assert.AnError)
		}()

		wg.Add(1)
		go func() {
			defer close(resultChan)
			defer wg.Done()

			<-ctx.Done()
		}()

		resultChan <- repository.GlobalConfigWatchResult{
			PrevState: config.GlobalConfig{},
			NewState:  config.GlobalConfig{},
			Err:       nil,
		}

		// when
		cancelFunc()

		wg.Wait()
	})
}

func Test_requirementsUpdater_triggerSync(t *testing.T) {
	t.Run("trigger fail on retrieving dogus", func(t *testing.T) {
		// given
		clientMock := testclient.NewClientBuilder().
			Build()

		sut := &RequirementsUpdater{
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

		generator := newMockRequirementsGenerator(t)
		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d1.Name).Return(nil, assert.AnError)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d2.Name).Return(nil, assert.AnError)
		localDoguRegMock.EXPECT().GetCurrent(mock.Anything, d3.Name).Return(nil, assert.AnError)

		sut := &RequirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
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

		generator := newMockRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &RequirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
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

		generator := newMockRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		testErr1 := errors.New("error1 occurred: wrong bitsize")
		testErr2 := errors.New("error2 occurred: out of entropy")
		testErr3 := errors.New("error3 failed to fail: bad luck")
		generator.EXPECT().Generate(testCtx, dj1).Return(v1.ResourceRequirements{}, testErr1)
		generator.EXPECT().Generate(testCtx, dj2).Return(v1.ResourceRequirements{}, testErr2)
		generator.EXPECT().Generate(testCtx, dj3).Return(v1.ResourceRequirements{}, testErr3)

		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &RequirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
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

		generator := newMockRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		generator.EXPECT().Generate(testCtx, dj1).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(testCtx, dj2).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(testCtx, dj3).Return(v1.ResourceRequirements{}, nil)

		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &RequirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
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

		generator := newMockRequirementsGenerator(t)
		dj1, dj2, dj3 := getTestDoguJsons()
		generator.EXPECT().Generate(testCtx, dj1).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(testCtx, dj2).Return(v1.ResourceRequirements{}, nil)
		generator.EXPECT().Generate(testCtx, dj3).Return(v1.ResourceRequirements{}, nil)

		localDoguRegMock := newMockDoguGetter(t)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d1.Name).Return(dj1, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d2.Name).Return(dj2, nil)
		localDoguRegMock.EXPECT().GetCurrent(testCtx, d3.Name).Return(dj3, nil)

		sut := &RequirementsUpdater{
			client:            clientMock,
			requirementsGen:   generator,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := sut.triggerSync(testCtx)

		// then
		require.NoError(t, err)
	})
}
