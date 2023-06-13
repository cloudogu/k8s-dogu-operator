package resource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

func TestNewUpserter(t *testing.T) {
	// given
	mockClient := extMocks.NewK8sClient(t)
	mockClient.On("Scheme").Return(new(runtime.Scheme))
	patcher := mocks.NewLimitPatcher(t)
	generator := extMocks.NewHostAliasGenerator(t)

	// when
	upserter := NewUpserter(mockClient, patcher, generator)

	// then
	require.NotNil(t, upserter)
	assert.Equal(t, mockClient, upserter.client)
	require.NotNil(t, upserter.generator)
}

func Test_upserter_updateOrInsert(t *testing.T) {
	t.Run("fail when using different types of objects", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		upserter := upserter{}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), nil, &appsv1.StatefulSet{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "upsert type must be a valid pointer to an K8s resource")
	})
	t.Run("should fail on incompatible input types", func(t *testing.T) {
		// given
		depl := &appsv1.Deployment{}
		svc := &v1.Service{}
		doguResource := readLdapDoguResource(t)
		sut := upserter{}

		// when
		err := sut.updateOrInsert(context.Background(), doguResource.GetObjectKey(), depl, svc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "incompatible types provided (*Deployment != *Service)")
	})

	t.Run("should update existing pcv when no controller reference is set", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		existingDeployment := readLdapDoguExpectedDeployment(t)
		// the test should override the replication count back to 1
		existingDeployment.Spec.Replicas = pointer.Int32(10)
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, existingDeployment).Build()
		upserter := upserter{client: testClient}

		// when
		upsertedDeployment := readLdapDoguExpectedDeployment(t)
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), &appsv1.Deployment{}, upsertedDeployment)

		// then
		require.NoError(t, err)

		afterUpsert := &appsv1.Deployment{}
		err = testClient.Get(context.Background(), doguResource.GetObjectKey(), afterUpsert)
		assert.Nil(t, afterUpsert.Spec.Replicas)
		// mock assert happens during cleanup
	})
}

func Test_upserter_UpsertDoguDeployment(t *testing.T) {
	ctx := context.Background()
	t.Run("fail on error when generating resource", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)
		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguDeployment", doguResource, dogu).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)
		mockClient.On("Get", ctx, doguResource.GetObjectKey(), &appsv1.Deployment{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguDeployment", doguResource, dogu).Return(readLdapDoguExpectedDeployment(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("successfully upsert deployment", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := mocks.NewDoguResourceGenerator(t)
		generatedDeployment := readLdapDoguExpectedDeployment(t)
		generator.On("CreateDoguDeployment", doguResource, dogu).Return(generatedDeployment, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}
		deploymentPatch := func(deployment *appsv1.Deployment) {
			deployment.Labels["test"] = "testvalue"
		}

		// when
		doguDeployment, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, deploymentPatch)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.ResourceVersion = "1"
		expectedDeployment.Labels["test"] = "testvalue"
		assert.Equal(t, expectedDeployment, doguDeployment)
	})
}

func Test_upserter_UpsertDoguPVCs(t *testing.T) {
	t.Run("fail when pvc already exists and retrier timeouts", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		dogu.Volumes = nil

		mockClient := extMocks.NewK8sClient(t)

		generator := mocks.NewDoguResourceGenerator(t)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}
		oldTries := maximumTriesWaitForExistingPVC
		maximumTriesWaitForExistingPVC = 2
		defer func() {
			maximumTriesWaitForExistingPVC = oldTries
		}()

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for existing pvc ldap-reserved to terminate: the maximum number of retries was reached: pvc ldap-reserved still exists")
	})

	t.Run("should throw an error if the resource generator fails to generate a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguPVC", doguResource).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to generate pvc")
	})

	t.Run("fail when upserting a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)

		mockClient.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.PersistentVolumeClaim{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguPVC", doguResource).Return(readLdapDoguExpectedDoguPVC(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("success when upserting a new dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := mocks.NewDoguResourceGenerator(t)
		expectedDoguPVC := readLdapDoguExpectedDoguPVC(t)
		generator.On("CreateDoguPVC", doguResource).Return(expectedDoguPVC, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualDoguPVC, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDoguPVC, actualDoguPVC)
	})

	t.Run("success when upserting a new dogu pvc when an old pvc is terminating", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		var reservedPvc *v1.PersistentVolumeClaim
		now := metav1.Now()
		reservedPvc = readLdapDoguExpectedDoguPVC(t)
		reservedPvc.DeletionTimestamp = &now
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, reservedPvc).Build()
		timer := time.NewTimer(time.Second * 5)
		go func() {
			<-timer.C
			err := testClient.Delete(context.Background(), reservedPvc)
			require.NoError(t, err)
		}()

		generator := mocks.NewDoguResourceGenerator(t)
		expectedDoguPVC := readLdapDoguExpectedDoguPVC(t)
		generator.On("CreateDoguPVC", doguResource).Return(expectedDoguPVC, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualDoguPVC, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDoguPVC, actualDoguPVC)
	})
}

func Test_upserter_UpsertDoguService(t *testing.T) {
	ctx := context.Background()
	t.Run("fail on error when generating resource", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := extMocks.NewK8sClient(t)
		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(ctx, doguResource, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := extMocks.NewK8sClient(t)
		mockClient.On("Get", ctx, doguResource.GetObjectKey(), &v1.Service{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		expectedService := readLdapDoguExpectedService(t)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(expectedService, nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(ctx, doguResource, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("successfully upsert service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		imageConfig := readLdapDoguImageConfig(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := mocks.NewDoguResourceGenerator(t)
		expectedService := readLdapDoguExpectedService(t)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(expectedService, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualService, err := upserter.UpsertDoguService(ctx, doguResource, imageConfig)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)
	})

	t.Run("fail when upserting a service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := extMocks.NewK8sClient(t)
		mockClient.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.Service{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(context.Background(), doguResource, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}
