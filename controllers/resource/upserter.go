package resource

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

const (
	errMsgFailedToGetPVC = "failed to get pvc"
)

var (
	noValidator                    cloudogu.ResourceValidator
	maximumTriesWaitForExistingPVC = 25
)

type upserter struct {
	client    thirdParty.K8sClient
	generator cloudogu.DoguResourceGenerator
}

// NewUpserter creates a new upserter that generates dogu resources and applies them to the cluster.
func NewUpserter(client client.Client, limitPatcher cloudogu.LimitPatcher, hostAliasGenerator thirdParty.HostAliasGenerator) *upserter {
	schema := client.Scheme()
	generator := NewResourceGenerator(schema, limitPatcher, hostAliasGenerator)
	return &upserter{client: client, generator: generator}
}

// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
// All parameters are mandatory except deploymentPatch which may be nil.
// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
func (u *upserter) UpsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, deploymentPatch func(*appsv1.Deployment)) (*appsv1.Deployment, error) {
	newDeployment, err := u.generator.CreateDoguDeployment(doguResource, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment: %w", err)
	}

	if deploymentPatch != nil {
		deploymentPatch(newDeployment)
	}

	err = u.updateOrInsert(ctx, doguResource.Name, doguResource.GetObjectKey(), &appsv1.Deployment{}, newDeployment, noValidator)
	if err != nil {
		return nil, err
	}

	return newDeployment, nil
}

// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
func (u *upserter) UpsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *imagev1.ConfigFile) (*v1.Service, error) {
	newService, err := u.generator.CreateDoguService(doguResource, image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.Name, doguResource.GetObjectKey(), &v1.Service{}, newService, noValidator)
	if err != nil {
		return nil, err
	}

	return newService, nil
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
	var serviceList []*v1.Service
	for _, newExposedService := range newExposedServices {
		exposedSvcKey := types.NamespacedName{
			Namespace: doguResource.GetNamespace(),
			Name:      newExposedService.Name,
		}
		err = u.updateOrInsert(ctx, doguResource.Name, exposedSvcKey, &v1.Service{}, newExposedService, noValidator)
		if err != nil {
			err2 := fmt.Errorf("failed to upsert exposed service %s: %w", newExposedService.ObjectMeta.Name, err)
			collectedErrs = multierror.Append(collectedErrs, err2)
			continue
		}

		serviceList = append(serviceList, newExposedService)
	}

	return serviceList, collectedErrs
}

// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
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
	pvcObjectKey := types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}

	actualPvc := &v1.PersistentVolumeClaim{}
	err := u.client.Get(ctx, pvcObjectKey, actualPvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return u.updateOrInsert(ctx, doguResource.Name, pvcObjectKey, &v1.PersistentVolumeClaim{}, pvc, &pvcValidator{})
		}

		return fmt.Errorf("%s %s: %w", errMsgFailedToGetPVC, pvcObjectKey.Name, err)
	}

	if actualPvc.DeletionTimestamp != nil {
		err = u.waitForExistingPVCToBeTerminated(ctx, pvcObjectKey)
		if err != nil {
			return fmt.Errorf("failed to wait for existing pvc %s to terminate: %w", pvc.Name, err)
		}

		return u.updateOrInsert(ctx, doguResource.Name, pvcObjectKey, &v1.PersistentVolumeClaim{}, pvc, &pvcValidator{})
	}

	// If the pvc exists and is not terminating keep it to support init data.
	return nil
}

func (u *upserter) waitForExistingPVCToBeTerminated(ctx context.Context, pvcObjectKey types.NamespacedName) error {
	err := retry.OnError(maximumTriesWaitForExistingPVC, pvcRetry, func() error {
		existingPVC := &v1.PersistentVolumeClaim{}
		err := u.client.Get(ctx, pvcObjectKey, existingPVC)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get pvc %s: %w", pvcObjectKey.Name, err)
		}

		log.FromContext(ctx).Info(fmt.Sprintf("wait for pvc %s to be terminated", pvcObjectKey.Name))
		return errors.New(fmt.Sprintf("pvc %s still exists", pvcObjectKey.Name))
	})

	return err
}

func pvcRetry(err error) bool {
	if strings.Contains(err.Error(), errMsgFailedToGetPVC) {
		return false
	}

	return true
}

func (u *upserter) updateOrInsert(ctx context.Context, doguName string, objectKey client.ObjectKey,
	resourceType client.Object, upsertResource client.Object, val cloudogu.ResourceValidator) error {
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

type pvcValidator struct{}

// Validate validates that a pvc contains all necessary data to be used as a valid dogu pvc.
func (v *pvcValidator) Validate(ctx context.Context, doguName string, resourceObj client.Object) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Starting validation of existing pvc in cluster with name [%s]", doguName))

	castedPVC, ok := resourceObj.(*v1.PersistentVolumeClaim)
	if !ok {
		return fmt.Errorf("unsupported validation object (expected: PVC): %v", resourceObj)
	}

	if castedPVC.Labels["dogu.name"] != doguName {
		return fmt.Errorf("pvc for dogu [%s] is not valid as pvc does not contain label [dogu.name] with value [%s]",
			doguName, doguName)
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
