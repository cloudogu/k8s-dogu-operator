package resource

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/log"

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

type resourceValidator interface {
	Validate(ctx context.Context, doguName string, obj client.Object) error
}

// doguResourceGenerator is used to generate kubernetes resources for the dogu.
type doguResourceGenerator interface {
	CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error)
	CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*v1.Service, error)
	CreateDoguPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
	CreateReservedPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
	CreateDoguExposedServices(doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]*v1.Service, error)
}

type upserter struct {
	client    client.Client
	generator doguResourceGenerator
}

// NewUpserter creates a new upserter that generates dogu resources and applies them to the cluster.
func NewUpserter(client client.Client, limitPatcher limitPatcher) *upserter {
	schema := client.Scheme()
	generator := NewResourceGenerator(schema, limitPatcher)
	return &upserter{client: client, generator: generator}
}

// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
// All parameters are mandatory except customDeployment which may be nil.
func (u *upserter) UpsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	newDeployment, err := u.generator.CreateDoguDeployment(doguResource, dogu, customDeployment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment: %w", err)
	}

	deployment := &appsv1.Deployment{}
	err = u.updateOrInsert(ctx, doguResource.Name, doguResource.GetObjectKey(), deployment, newDeployment, noValidator)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
func (u *upserter) UpsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *imagev1.ConfigFile) (*v1.Service, error) {
	newService, err := u.generator.CreateDoguService(doguResource, image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service: %w", err)
	}

	service := &v1.Service{}
	err = u.updateOrInsert(ctx, doguResource.Name, doguResource.GetObjectKey(), service, newService, noValidator)
	if err != nil {
		return nil, err
	}

	return service, nil
}

// UpsertDoguExposedServices creates exposed services based on the given dogu. If an error occurs during creating
// several exposed services, this method tries to apply as many exposed services as possible and returns then
// an error collection.
func (u *upserter) UpsertDoguExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]*v1.Service, error) {
	newExposedServices, err := u.generator.CreateDoguExposedServices(doguResource, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate exposed services: %w", err)
	}

	var collectedErrs error
	serviceList := []*v1.Service{}
	for _, newExposedService := range newExposedServices {
		service := &v1.Service{}
		err = u.updateOrInsert(ctx, doguResource.Name, doguResource.GetObjectKey(), service, newExposedService, noValidator)
		if err != nil {
			err2 := fmt.Errorf("failed to upsert exposed service %s: %w", newExposedService.ObjectMeta.Name, err)
			collectedErrs = multierror.Append(collectedErrs, err2)
			continue
		}

		serviceList = append(serviceList, service)
	}

	return serviceList, collectedErrs
}

// UpsertDoguPVCs generates a persitent volume claim for a given dogu and applies it to the cluster.
func (u *upserter) UpsertDoguPVCs(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (*v1.PersistentVolumeClaim, error) {
	newReservedPVC, err := u.generator.CreateReservedPVC(doguResource)
	if err != nil {
		return nil, err
	}

	err = u.upsertPVC(ctx, newReservedPVC, doguResource)
	if err != nil {
		return nil, err
	}

	if len(dogu.Volumes) > 0 {
		newPVC, err := u.generator.CreateDoguPVC(doguResource)
		if err != nil {
			return nil, fmt.Errorf("failed to generate pvc: %w", err)
		}

		err = u.upsertPVC(ctx, newPVC, doguResource)
		if err != nil {
			return nil, err
		}

		return newPVC, nil
	}

	return nil, nil
}

func (u *upserter) upsertPVC(ctx context.Context, pvc *v1.PersistentVolumeClaim, doguResource *k8sv1.Dogu) error {
	existingPVC := &v1.PersistentVolumeClaim{}
	pvcObjectKey := types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}

	err := u.client.Get(ctx, pvcObjectKey, existingPVC)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	objectKey := client.ObjectKey{Name: pvc.Name, Namespace: pvc.Namespace}
	// Only create a PVC but do not update it (for now) because updating immutable PVCs is tough
	if apierrors.IsNotFound(err) {
		err = u.updateOrInsert(ctx, doguResource.Name, objectKey, &v1.PersistentVolumeClaim{}, pvc, &longhornPVCValidator{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *upserter) updateOrInsert(ctx context.Context, doguName string, objectKey client.ObjectKey,
	resourceType client.Object, upsertResource client.Object, val resourceValidator) error {
	if resourceType == nil {
		return errors.New("upsert type must be a valid pointer to an K8s resource")
	}
	ok, type1, type2 := sameResourceTypes(resourceType, upsertResource)
	if !ok {
		return fmt.Errorf("incompatible types provided (%s != %s)", type1, type2)
	}

	err := u.client.Get(ctx, objectKey, resourceType)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if apierrors.IsNotFound(err) {
		return u.client.Create(ctx, upsertResource)
	}

	// use resourceType here because it origins from the cluster state while upsertResource was artificially created so
	// it does not contain any useful metadata.
	ownerRef := metav1.GetControllerOf(resourceType)
	if ownerRef != nil && val != nil {
		err = val.Validate(ctx, doguName, resourceType)
		if err != nil {
			return err
		}
		// update existing resource either way
	}

	return u.client.Update(ctx, upsertResource)
}

type longhornPVCValidator struct{}

// Validate validates that a pvc contains all necessary data to be used as a valid dogu pvc.
func (v *longhornPVCValidator) Validate(ctx context.Context, doguName string, resourceObj client.Object) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Starting validation of existing pvc in cluster with name [%s]", doguName))

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

func sameResourceTypes(resourceType client.Object, newResource client.Object) (bool, string, string) {
	if reflect.TypeOf(resourceType).AssignableTo(reflect.TypeOf(newResource)) {
		return true, "", ""
	}

	return false, getTypeName(resourceType), getTypeName(newResource)
}

func getTypeName(objectInQuestion interface{}) string {
	// we don't check if the object is of pointer type because the method signature of updateOrInsert enforces this for us
	t := reflect.TypeOf(objectInQuestion)
	return "*" + t.Elem().Name()
}
