package resource

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/api/v1/mocks"
	mocks2 "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewUpserter(t *testing.T) {
	// given
	client := &mocks.Client{}
	client.On("Scheme").Return(new(runtime.Scheme))
	patcher := &mocks2.LimitPatcher{}

	// when
	upserter := NewUpserter(client, patcher)

	// given
	require.NotNil(t, upserter)
	assert.Equal(t, client, upserter.client)
	require.NotNil(t, upserter.generator)
}

func Test_longhornPVCValidator_validate(t *testing.T) {
	t.Run("error on validating pvc with non pvc object", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testObject := &appsv1.Deployment{}

		// when
		err := validator.Validate(context.Background(), "name", testObject)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported validation object (expected: PVC)")
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

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as annotation [volume.beta.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
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

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as annotation [volume.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
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

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as pvc does not contain label [dogu] with value [name]")
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

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as pvc has invalid storage class: the storage class must be [longhorn]")
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

		// given
		require.NoError(t, err)
	})
}

func Test_upserter_ApplyDoguResource(t *testing.T) {
	ctx := context.Background()

	t.Run("fail on error when creating deployment", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(nil, assert.AnError)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, nil)

		// given
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to generate deployment")
		mock.AssertExpectationsForObjects(t, generator)
	})

	t.Run("fail on error when creating service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(readLdapDoguExpectedDeployment(t), nil)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(nil, assert.AnError)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, nil)

		// given
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to generate service")
		mock.AssertExpectationsForObjects(t, generator)
	})

	t.Run("fail on error when creating exposed services", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(readLdapDoguExpectedDeployment(t), nil)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(nil, assert.AnError)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, nil)

		// given
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to generate exposed services")
		mock.AssertExpectationsForObjects(t, generator)
	})

	t.Run("fail on error when creating pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(readLdapDoguExpectedDeployment(t), nil)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(readLdapDoguExpectedExposedServices(t), nil)
		generator.On("CreateDoguPVC", doguResource).Return(nil, assert.AnError)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, nil)

		// given
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to generate pvc")
		mock.AssertExpectationsForObjects(t, generator)
	})

	t.Run("success in creating new objects", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(readLdapDoguExpectedDeployment(t), nil)
		generator.On("CreateDoguService", doguResource, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(readLdapDoguExpectedExposedServices(t), nil)
		generator.On("CreateDoguPVC", doguResource).Return(readLdapDoguExpectedPVC(t), nil)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, nil)

		// given
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, generator)
	})
}

func Test_upserter_updateOrInsert(t *testing.T) {
	t.Run("fail when using different types of objects", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		upserter := upserter{}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), nil, &appsv1.StatefulSet{}, noValidator)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upsert type must be a valid pointer to an K8s resource")
	})

	t.Run("update existing pcv when no controller reference is set and fail on validation", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, readLdapDoguExpectedDeployment(t)).Build()
		resourceValidator := &mocks2.ResourceValidatorMock{}
		resourceValidator.On("Validate", context.Background(), doguResource.Name, mock.Anything).Return(assert.AnError)
		upserter := upserter{client: client}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), &appsv1.Deployment{}, readLdapDoguExpectedDeployment(t), resourceValidator)

		// given
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("update existing pcv when no controller reference is set and validation works", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		existingDeployment := readLdapDoguExpectedDeployment(t)
		// the test should override the replication count back to 1
		existingDeployment.Spec.Replicas = pointer.Int32(10)
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, existingDeployment).Build()
		resourceValidator := &mocks2.ResourceValidatorMock{}
		resourceValidator.On("Validate", context.Background(), doguResource.Name, mock.Anything).Return(nil)
		upserter := upserter{client: client}

		// when
		upsertedDeployment := readLdapDoguExpectedDeployment(t)
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), &appsv1.Deployment{}, upsertedDeployment, resourceValidator)

		// given
		require.NoError(t, err)

		afterUpsert := &appsv1.Deployment{}
		err = client.Get(context.Background(), doguResource.GetObjectKey(), afterUpsert)
		assert.Nil(t, afterUpsert.Spec.Replicas)
	})
}

func Test_upserter_upsertDoguDeployment(t *testing.T) {
	t.Run("fail when upserting a deployment", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		client := &mocks.Client{}
		client.On("Get", context.Background(), doguResource.GetObjectKey(), &appsv1.Deployment{}).Return(assert.AnError)

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguDeployment", doguResource, dogu, mock.AnythingOfType("*v1.Deployment")).Return(readLdapDoguExpectedDeployment(t), nil)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.upsertDoguDeployment(context.Background(), doguResource, dogu, nil)

		// given
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generator, client)
	})
}

func Test_upserter_upsertDoguExposedServices(t *testing.T) {
	t.Run("fail when upserting a exposed services", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		client := &mocks.Client{}
		failedToCreateFirstError := errors.New("failed on exposed service 1")
		client.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.Service{}).Once().Return(failedToCreateFirstError)
		failedToCreateSecondError := errors.New("failed on exposed service 2")
		client.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.Service{}).Once().Return(failedToCreateSecondError)

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguExposedServices", doguResource, dogu).Return(readLdapDoguExpectedExposedServices(t), nil)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.upsertDoguExposedServices(context.Background(), doguResource, dogu)

		// given
		multiError := new(multierror.Error)
		require.ErrorAs(t, err, &multiError)
		require.ErrorIs(t, multiError.Errors[0], failedToCreateFirstError)
		assert.Contains(t, multiError.Errors[0].Error(), "failed to upsert exposed service ldap-exposed-2222")
		require.ErrorIs(t, multiError.Errors[1], failedToCreateSecondError)
		assert.Contains(t, multiError.Errors[1].Error(), "failed to upsert exposed service ldap-exposed-8888")
		mock.AssertExpectationsForObjects(t, generator, client)
	})
}

func Test_upserter_upsertDoguPVC(t *testing.T) {
	t.Run("fail when upserting a pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)

		client := &mocks.Client{}
		client.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.PersistentVolumeClaim{}).Return(assert.AnError)

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguPVC", doguResource).Return(readLdapDoguExpectedPVC(t), nil)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.upsertDoguPVC(context.Background(), doguResource)

		// given
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generator, client)
	})
}

func Test_upserter_upsertDoguService(t *testing.T) {
	t.Run("fail when upserting a service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		imageConfig := readLdapDoguImageConfig(t)

		client := &mocks.Client{}
		client.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.Service{}).Return(assert.AnError)

		generator := &mocks2.DoguResourceGenerator{}
		generator.On("CreateDoguService", doguResource, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		upserter := upserter{
			client:    client,
			generator: generator,
		}

		// when
		err := upserter.upsertDoguService(context.Background(), doguResource, imageConfig)

		// given
		require.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, generator, client)
	})
}
