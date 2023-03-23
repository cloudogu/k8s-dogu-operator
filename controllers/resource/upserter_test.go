package resource

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

func TestNewUpserter(t *testing.T) {
	// given
	mockClient := extMocks.NewK8sClient(t)
	mockClient.On("Scheme").Return(new(runtime.Scheme))
	patcher := mocks.NewLimitPatcher(t)
	generator := mocks.NewHostAliasGenerator(t)

	// when
	upserter := NewUpserter(mockClient, patcher, generator)

	// then
	require.NotNil(t, upserter)
	assert.Equal(t, mockClient, upserter.client)
	require.NotNil(t, upserter.generator)
}

func Test_longhornPVCValidator_validate(t *testing.T) {
	t.Run("error on validating pvc with non pvc object", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testObject := &appsv1.Deployment{}

		// when
		err := validator.Validate(context.Background(), "name", testObject)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported validation object (expected: PVC)")
	})

	t.Run("error on missing beta longhorn annotation", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		}

		// when
		err := validator.Validate(context.Background(), "name", testPvc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc for dogu [name] is not valid as annotation [volume.beta.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
	})
	t.Run("error on missing default longhorn annotation", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
				},
			},
		}

		// when
		err := validator.Validate(context.Background(), "name", testPvc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc for dogu [name] is not valid as annotation [volume.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
	})

	t.Run("error on missing dogu label", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
			},
		}

		// when
		err := validator.Validate(context.Background(), "name", testPvc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc for dogu [name] is not valid as pvc does not contain label [dogu] with value [name]")
	})

	t.Run("error on missing dogu label", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
				Labels: map[string]string{"dogu": "name"},
			},
			Spec: v1.PersistentVolumeClaimSpec{StorageClassName: pointer.String("invalidStorageClass")},
		}

		// when
		err := validator.Validate(context.Background(), "name", testPvc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc for dogu [name] is not valid as pvc has invalid storage class: the storage class must be [longhorn]")
	})

	t.Run("success", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
				Labels: map[string]string{"dogu": "name"},
			},
			Spec: v1.PersistentVolumeClaimSpec{StorageClassName: pointer.String(longhornStorageClassName)},
		}

		// when
		err := validator.Validate(context.Background(), "name", testPvc)

		// then
		require.NoError(t, err)
	})
}

func Test_upserter_updateOrInsert(t *testing.T) {
	t.Run("fail when using different types of objects", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		upserter := upserter{}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.Name, doguResource.GetObjectKey(), nil, &appsv1.StatefulSet{}, noValidator)

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
		err := sut.updateOrInsert(context.Background(), doguResource.Name, doguResource.GetObjectKey(), depl, svc, noValidator)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "incompatible types provided (*Deployment != *Service)")
	})
	t.Run("should update existing pcv when no controller reference is set and fail on validation", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, readLdapDoguExpectedDeployment(t)).Build()
		resourceValidator := mocks.NewResourceValidator(t)
		resourceValidator.On("Validate", context.Background(), doguResource.Name, mock.Anything).Return(assert.AnError)
		upserter := upserter{client: testClient}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.Name, doguResource.GetObjectKey(), &appsv1.Deployment{}, readLdapDoguExpectedDeployment(t), resourceValidator)

		// then
		require.ErrorIs(t, err, assert.AnError)
		// mock assert happens during cleanup
	})

	t.Run("should update existing pcv when no controller reference is set and validation works", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		existingDeployment := readLdapDoguExpectedDeployment(t)
		// the test should override the replication count back to 1
		existingDeployment.Spec.Replicas = pointer.Int32(10)
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, existingDeployment).Build()
		resourceValidator := mocks.NewResourceValidator(t)
		resourceValidator.On("Validate", context.Background(), doguResource.Name, mock.Anything).Return(nil)
		upserter := upserter{client: testClient}

		// when
		upsertedDeployment := readLdapDoguExpectedDeployment(t)
		err := upserter.updateOrInsert(context.Background(), doguResource.Name, doguResource.GetObjectKey(), &appsv1.Deployment{}, upsertedDeployment, resourceValidator)

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

func Test_upserter_UpsertDoguExposedServices(t *testing.T) {
	t.Run("fail when generating exposed services", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)
		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguExposedServices(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting a exposed services", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)
		failedToCreateFirstError := errors.New("failed on exposed service 1")
		mockClient.On("Get", context.Background(), types.NamespacedName{Namespace: "ecosystem", Name: "ldap-exposed-2222"}, &v1.Service{}).Once().Return(failedToCreateFirstError)
		failedToCreateSecondError := errors.New("failed on exposed service 2")
		mockClient.On("Get", context.Background(), types.NamespacedName{Namespace: "ecosystem", Name: "ldap-exposed-8888"}, &v1.Service{}).Once().Return(failedToCreateSecondError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(readLdapDoguExpectedExposedServices(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguExposedServices(context.Background(), doguResource, dogu)

		// then
		multiError := new(multierror.Error)
		require.ErrorAs(t, err, &multiError)
		require.ErrorIs(t, multiError.Errors[0], failedToCreateFirstError)
		assert.Contains(t, multiError.Errors[0].Error(), "failed to upsert exposed service ldap-exposed-2222")
		require.ErrorIs(t, multiError.Errors[1], failedToCreateSecondError)
		assert.Contains(t, multiError.Errors[1].Error(), "failed to upsert exposed service ldap-exposed-8888")
	})
	t.Run("successfully create exposed services", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := mocks.NewDoguResourceGenerator(t)
		expectedExposedServices := readLdapDoguExpectedExposedServices(t)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(expectedExposedServices, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualExposedServices, err := upserter.UpsertDoguExposedServices(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedExposedServices, actualExposedServices)
	})
}

func Test_upserter_UpsertDoguPVCs(t *testing.T) {
	t.Run("fail when creating a reserved pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		dogu.Volumes = nil

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(nil, assert.AnError)
		upserter := upserter{
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting a reserved pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		dogu.Volumes = nil

		mockClient := extMocks.NewK8sClient(t)
		key := doguResource.GetObjectKey()
		key.Name = doguResource.GetReservedPVCName()
		mockClient.On("Get", mock.Anything, key, &v1.PersistentVolumeClaim{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("fail when pvc already exists and retrier timeouts", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		dogu.Volumes = nil

		mockClient := extMocks.NewK8sClient(t)
		key := doguResource.GetObjectKey()
		key.Name = doguResource.GetReservedPVCName()
		mockClient.EXPECT().Get(mock.Anything, key, &v1.PersistentVolumeClaim{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			pvc := object.(*v1.PersistentVolumeClaim)
			now := metav1.Now()
			pvc.SetDeletionTimestamp(&now)

			return nil
		})

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
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

	resourceNotFoundError := apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, "")

	t.Run("should throw an error if the resource generator fails to generate a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)

		key := doguResource.GetObjectKey()
		key.Name = doguResource.GetReservedPVCName()
		mockClient.On("Get", mock.Anything, key, &v1.PersistentVolumeClaim{}).Return(resourceNotFoundError)
		mockClient.On("Create", mock.Anything, readLdapDoguExpectedReservedPVC(t)).Return(nil)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
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

	t.Run("should throw an error if the resource generator fails to generate a dogu reserved pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)

		key := doguResource.GetObjectKey()
		key.Name = doguResource.GetReservedPVCName()
		mockClient.On("Get", mock.Anything, key, &v1.PersistentVolumeClaim{}).Return(resourceNotFoundError)
		mockClient.On("Create", mock.Anything, readLdapDoguExpectedReservedPVC(t)).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("fail when upserting a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := extMocks.NewK8sClient(t)

		key := doguResource.GetObjectKey()
		key.Name = doguResource.GetReservedPVCName()
		mockClient.On("Get", context.Background(), key, &v1.PersistentVolumeClaim{}).Return(resourceNotFoundError)
		mockClient.On("Create", mock.Anything, readLdapDoguExpectedReservedPVC(t)).Return(nil)
		mockClient.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.PersistentVolumeClaim{}).Return(assert.AnError)

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
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
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
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
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
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

	t.Run("success when only creating reserved pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		dogu.Volumes = nil

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := mocks.NewDoguResourceGenerator(t)
		generator.On("CreateReservedPVC", doguResource).Return(readLdapDoguExpectedReservedPVC(t), nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualDoguPVC, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Nil(t, actualDoguPVC)
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
