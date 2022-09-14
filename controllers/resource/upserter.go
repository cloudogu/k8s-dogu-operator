package resource

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKubernetesVolumeDriver     = "volume.kubernetes.io/storage-provisioner"
	annotationKubernetesBetaVolumeDriver = "volume.beta.kubernetes.io/storage-provisioner"
	longhornDiverID                      = "driver.longhorn.io"
	longhornStorageClassName             = "longhorn"
)

var noValidator resourceValidator

// NewUpserter creates a new upserter that generates dogu resources and applies them to the cluster.
func NewUpserter(client client.Client, limitPatcher limitPatcher) *upserter {
	schema := client.Scheme()
	generator := NewResourceGenerator(schema, limitPatcher)
	return &upserter{client: client, generator: generator}
}

type resourceValidator interface {
	validate(ctx context.Context, doguName string, obj client.Object) error
}

type upserter struct {
	client    client.Client
	generator *ResourceGenerator
}

// ApplyDoguResource generates K8s resources from a given dogu and applies them to the cluster.
// All parameters are mandatory except customDeployment which may be nil.
func (u *upserter) ApplyDoguResource(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error {
	err := u.upsertDoguDeployment(ctx, doguResource, dogu, customDeployment)
	if err != nil {
		return err
	}

	err = u.upsertDoguService(ctx, doguResource, image)
	if err != nil {
		return err
	}

	err = u.upsertDoguExposedServices(ctx, doguResource, dogu)
	if err != nil {
		return err
	}

	err = u.upsertDoguPVC(ctx, doguResource)
	if err != nil {
		return err
	}

	return nil
}

func (u *upserter) upsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) error {
	newDeployment, err := u.generator.CreateDoguDeployment(doguResource, dogu, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to generate deployment: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &appsv1.Deployment{}, newDeployment, noValidator)
	if err != nil {
		return err
	}

	return nil
}

func (u *upserter) upsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *imagev1.ConfigFile) error {
	newService, err := u.generator.CreateDoguService(doguResource, image)
	if err != nil {
		return fmt.Errorf("failed to generate service: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &v1.Service{}, newService, noValidator)
	if err != nil {
		return err
	}

	return nil
}

// upsertDoguExposedServices creates exposed services based on the given dogu. If an error occurs during creating
// several exposed services, this method tries to apply as many exposed services as possible and returns then
// an error collection.
func (u *upserter) upsertDoguExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	newExposedServices, err := u.generator.CreateDoguExposedServices(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to generate service: %w", err)
	}

	var collectedErrs error

	for _, newExposedService := range newExposedServices {
		err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &v1.Service{}, newExposedService, noValidator)
		if err != nil {
			err2 := fmt.Errorf("failed to upsert exposed service %s: %w", newExposedService.ObjectMeta.Name, err)
			collectedErrs = multierror.Append(collectedErrs, err2)
		}
	}

	return collectedErrs
}

func (u *upserter) upsertDoguPVC(ctx context.Context, doguResource *k8sv1.Dogu) error {
	newPVC, err := u.generator.CreateDoguPVC(doguResource)
	if err != nil {
		return fmt.Errorf("failed to generate service: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &v1.PersistentVolumeClaim{}, newPVC, &longhornPVCValidator{})
	if err != nil {
		return err
	}

	return nil
}

func (u *upserter) updateOrInsert(ctx context.Context, objectKey *client.ObjectKey, resourceType client.Object, newResource client.Object, val resourceValidator) error {
	if resourceType == nil {
		return errors.New("upsert type must be a valid pointer to an K8s resource")
	}

	err := u.client.Get(ctx, *objectKey, resourceType)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if apierrors.IsNotFound(err) {
		return u.client.Create(ctx, newResource)
	}

	// use resourceType here because it origins from the cluster state while newResource was artificially created so
	// it does not contain any useful metadata.
	ownerRef := metav1.GetControllerOf(resourceType)
	if ownerRef != nil && val != nil {
		err = val.validate(ctx, objectKey.Name, resourceType)
		if err != nil {
			return err
		}
		// update existing resource either way
	}

	return u.client.Update(ctx, newResource)
}

type longhornPVCValidator struct{}

func (v *longhornPVCValidator) validate(ctx context.Context, doguName string, resourceObj client.Object) error {
	castedPVC, ok := resourceObj.(*v1.PersistentVolumeClaim)
	if !ok {
		return fmt.Errorf("unsupported validation object (expected: PVC): %v", resourceObj)
	}

	if castedPVC.Annotations[annotationKubernetesBetaVolumeDriver] != longhornDiverID {
		return fmt.Errorf("pvc for dogu [%s] is not valid as annotation [%s] does not exist or is not [%s]",
			doguName, annotationKubernetesBetaVolumeDriver, longhornDiverID)
	}

	if castedPVC.Annotations[annotationKubernetesVolumeDriver] != longhornDiverID {
		return fmt.Errorf("pvc for dogu [%s] is not valid as annotation [%s] does not exist or is not [%s]",
			doguName, annotationKubernetesVolumeDriver, longhornDiverID)
	}

	if castedPVC.Labels["dogu"] != doguName {
		return fmt.Errorf("pvc for dogu [%s] is not valid as pvc does not contain label [dogu] with value [%s]",
			doguName, doguName)
	}

	if *castedPVC.Spec.StorageClassName != longhornStorageClassName {
		return fmt.Errorf("pvc for dogu [%s] is not valid as pvc has invalid storage class: the storage class must be [%s]",
			doguName, longhornStorageClassName)
	}

	return nil
}
