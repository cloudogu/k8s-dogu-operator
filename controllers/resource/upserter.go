package resource

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/retry-lib/retry"
)

const (
	errMsgFailedToGetPVC    = "failed to get pvc"
	k8sNginxIngressDoguName = "nginx-ingress"
	k8sNginxStaticDoguName  = "nginx-static"
)

var (
	maximumTriesWaitForExistingPVC = 25
)

type upserter struct {
	client           k8sClient
	generator        DoguResourceGenerator
	exposedPortAdder exposePortAdder
}

// NewUpserter creates a new upserter that generates dogu resources and applies them to the cluster.
func NewUpserter(client client.Client, generator DoguResourceGenerator) *upserter {
	exposedPortAdder := NewDoguExposedPortHandler(client)
	return &upserter{
		client:           client,
		generator:        generator,
		exposedPortAdder: exposedPortAdder,
	}
}

// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
// All parameters are mandatory except deploymentPatch which may be nil.
// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
func (u *upserter) UpsertDoguDeployment(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, deploymentPatch func(*appsv1.Deployment)) (*appsv1.Deployment, error) {
	newDeployment, err := u.generator.CreateDoguDeployment(doguResource, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment: %w", err)
	}

	if deploymentPatch != nil {
		deploymentPatch(newDeployment)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &appsv1.Deployment{}, newDeployment)
	if err != nil {
		return nil, err
	}

	return newDeployment, nil
}

// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
func (u *upserter) UpsertDoguService(ctx context.Context, doguResource *k8sv2.Dogu, image *imagev1.ConfigFile) (*v1.Service, error) {
	newService, err := u.generator.CreateDoguService(doguResource, image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &v1.Service{}, newService)
	if err != nil {
		return nil, err
	}

	return newService, nil
}

func (u *upserter) UpsertDoguExposedService(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*v1.Service, error) {
	return u.exposedPortAdder.CreateOrUpdateCesLoadbalancerService(ctx, doguResource, dogu)
}

// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
func (u *upserter) UpsertDoguPVCs(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*v1.PersistentVolumeClaim, error) {
	shouldCreatePVC := false
	for _, volume := range dogu.Volumes {
		if volume.NeedsBackup {
			shouldCreatePVC = true
			break
		}
	}

	if shouldCreatePVC {
		newPVC, err := u.generator.CreateDoguPVC(doguResource)
		if err != nil {
			return nil, fmt.Errorf("failed to generate pvc: %w", err)
		}

		err = u.upsertPVC(ctx, newPVC)
		if err != nil {
			return nil, err
		}

		return newPVC, nil
	}

	return nil, nil
}

// UpsertDoguNetworkPolicies generates the network policies for a dogu
func (u *upserter) UpsertDoguNetworkPolicies(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	var multiErr error
	denyAllPolicy := generateDenyAllPolicy(doguResource, dogu)

	if err := u.updateOrInsert(ctx, getNetPolObjectKey(denyAllPolicy), &netv1.NetworkPolicy{}, denyAllPolicy); err != nil {
		errors.Join(multiErr, fmt.Errorf("failed to create or update deny all rule for dogu %s: %w", dogu.GetSimpleName(), err))
	}

	for _, dependency := range dogu.Dependencies {
		if dependency.Type == "dogu" {
			dependencyName := dependency.Name
			if dependencyName == k8sNginxStaticDoguName {
				continue
			}

			if dependencyName == k8sNginxIngressDoguName {
				ingressPolicy := generateIngressNetPol(doguResource, dogu)

				if err := u.updateOrInsert(ctx, getNetPolObjectKey(ingressPolicy), &netv1.NetworkPolicy{}, ingressPolicy); err != nil {
					errors.Join(multiErr, fmt.Errorf("failed to create or update ingress network policy for dogu %s: %w", dogu.GetSimpleName(), err))
				}

				continue
			}

			dependencyPolicy := generateDoguDepNetPol(doguResource, dogu, dependencyName)
			if err := u.updateOrInsert(ctx, getNetPolObjectKey(dependencyPolicy), &netv1.NetworkPolicy{}, dependencyPolicy); err != nil {
				errors.Join(multiErr, fmt.Errorf("failed to create or update network policy allow rule for dependency %s of dogu %s: %w", dependencyName, dogu.GetSimpleName(), err))
			}
		}
	}

	if multiErr != nil {
		logger := log.FromContext(ctx)
		logger.Error(multiErr, fmt.Sprintf("failed to create some network policies for dogu %s", dogu.GetSimpleName()))
		return multiErr
	}

	return nil
}

func (u *upserter) upsertPVC(ctx context.Context, pvc *v1.PersistentVolumeClaim) error {
	pvcObjectKey := types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}

	actualPvc := &v1.PersistentVolumeClaim{}
	err := u.client.Get(ctx, pvcObjectKey, actualPvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return u.updateOrInsert(ctx, pvcObjectKey, &v1.PersistentVolumeClaim{}, pvc)
		}

		return fmt.Errorf("%s %s: %w", errMsgFailedToGetPVC, pvcObjectKey.Name, err)
	}

	if actualPvc.DeletionTimestamp != nil {
		err = u.waitForExistingPVCToBeTerminated(ctx, pvcObjectKey)
		if err != nil {
			return fmt.Errorf("failed to wait for existing pvc %s to terminate: %w", pvc.Name, err)
		}

		return u.updateOrInsert(ctx, pvcObjectKey, &v1.PersistentVolumeClaim{}, pvc)
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
		return fmt.Errorf("pvc %s still exists", pvcObjectKey.Name)
	})

	return err
}

func pvcRetry(err error) bool {
	return !strings.Contains(err.Error(), errMsgFailedToGetPVC)
}

func (u *upserter) updateOrInsert(ctx context.Context, objectKey client.ObjectKey,
	resourceType client.Object, upsertResource client.Object) error {
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

	return u.client.Update(ctx, upsertResource)
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
