package resource

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"

	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/retry-lib/retry"
)

const (
	errMsgFailedToGetPVC        = "failed to get pvc"
	k8sCesGatewayComponentLabel = "k8s.cloudogu.com/component.name"
	k8sCesGatewayComponentName  = "k8s-ces-gateway"
	dependencyTypeDogu          = "dogu"
	dependencyTypeComponent     = "component"
)

var (
	maximumTriesWaitForExistingPVC = 25
)

type upserter struct {
	client                 k8sClient
	generator              DoguResourceGenerator
	networkPoliciesEnabled bool
}

// NewUpserter creates a new upserter that generates dogu resources and applies them to the cluster.
func NewUpserter(client client.Client, generator DoguResourceGenerator, config *opConfig.OperatorConfig) ResourceUpserter {
	return &upserter{
		client:                 client,
		generator:              generator,
		networkPoliciesEnabled: config.NetworkPoliciesEnabled,
	}
}

// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
// All parameters are mandatory except deploymentPatch which may be nil.
// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
func (u *upserter) UpsertDoguDeployment(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, deploymentPatch func(*appsv1.Deployment)) (*appsv1.Deployment, error) {
	newDeployment, err := u.generator.CreateDoguDeployment(ctx, doguResource, dogu)
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
func (u *upserter) UpsertDoguService(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, image *imagev1.ConfigFile) (*v1.Service, error) {
	newService, err := u.generator.CreateDoguService(doguResource, dogu, image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service: %w", err)
	}

	err = u.updateOrInsert(ctx, doguResource.GetObjectKey(), &v1.Service{}, newService)
	if err != nil {
		return nil, err
	}

	return newService, nil
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
func (u *upserter) UpsertDoguNetworkPolicies(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, service *v1.Service) error {
	logger := log.FromContext(ctx)
	if !u.networkPoliciesEnabled {
		logger.Info("Do not create network policies as they are disabled by configuration; deleting previously applied network policies")

		err := u.client.DeleteAllOf(ctx,
			&netv1.NetworkPolicy{},
			client.InNamespace(doguResource.Namespace),
			client.MatchingLabels{k8sv2.DoguLabelName: dogu.GetSimpleName()},
		)
		if err != nil {
			return fmt.Errorf("failed to delete network policies because they are disabled: %w", err)
		}

		return nil
	}

	var multiErr error
	denyAllPolicy := generateDenyAllPolicy(doguResource, dogu)

	if err := u.upsertNetworkPolicy(ctx, denyAllPolicy); err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("failed to create or update deny all rule for dogu %s: %w", dogu.GetSimpleName(), err))
	}

	var allDependencies = append(dogu.Dependencies, dogu.OptionalDependencies...)

	err := u.upsertNetworkPoliciesForDependencies(ctx, doguResource, dogu, allDependencies)
	if err != nil {
		multiErr = errors.Join(multiErr, err)
	}

	if err := u.upsertServiceAnnotationNetworkPolicy(ctx, doguResource, dogu, service); err != nil {
		multiErr = errors.Join(multiErr, err)
	}

	if err := u.deleteNonExistentDependencyPolicies(ctx, dogu, allDependencies); err != nil {
		multiErr = errors.Join(multiErr, err)
	}

	if multiErr != nil {
		logger := log.FromContext(ctx)
		logger.Error(multiErr, fmt.Sprintf("failed to create some network policies for dogu %s", dogu.GetSimpleName()))
		return multiErr
	}

	return nil
}

func (u *upserter) upsertNetworkPoliciesForDependencies(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, allDependencies []core.Dependency) error {
	var multiErr error
	for _, dependency := range allDependencies {
		if dependency.Type == dependencyTypeDogu {
			if err := u.upsertDoguDependencyNetworkPolicy(ctx, dependency.Name, doguResource, dogu); err != nil {
				multiErr = errors.Join(multiErr, err)
			}
		}
		if dependency.Type == dependencyTypeComponent {
			if err := u.upsertComponentDependencyNetworkPolicy(ctx, dependency.Name, doguResource, dogu); err != nil {
				multiErr = errors.Join(multiErr, err)
			}
		}
	}
	return multiErr
}

func (u *upserter) deleteNonExistentDependencyPolicies(ctx context.Context, dogu *core.Dogu, allDependencies []core.Dependency) error {
	var multiErr error

	currentPolicies := &netv1.NetworkPolicyList{}
	if err := u.client.List(ctx, currentPolicies, client.MatchingLabels{"dogu.name": dogu.GetSimpleName()}); err != nil {
		return err
	}

	for _, policy := range currentPolicies.Items {
		// Only delete policies which rely on a dependency
		if policy.Labels[depenendcyLabel] == "" {
			continue
		}

		// Check if the dependency of the network policy still exists in dogu dependencies
		stillExists := slices.ContainsFunc(allDependencies, func(dependency core.Dependency) bool {
			return dependency.Name == policy.Labels[depenendcyLabel]
		})

		// If not existent in dogu dependency, remove the network policy
		if !stillExists {
			if err := u.client.Delete(ctx, &policy); err != nil {
				multiErr = errors.Join(multiErr, err)
			}
		}
	}

	return multiErr
}

func (u *upserter) upsertServiceAnnotationNetworkPolicy(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, service *v1.Service) error {
	if _, ok := service.Annotations[annotation.CesServicesAnnotation]; !ok {
		return nil
	}
	dependencyNetworkPolicy := generateIngressNetPol(doguResource, dogu)
	if err := u.upsertNetworkPolicy(ctx, dependencyNetworkPolicy); err != nil {
		return fmt.Errorf("failed to create or update network policy allow rule for ingress of dogu %s: %w", dogu.GetSimpleName(), err)
	}
	return nil
}

func (u *upserter) upsertDoguDependencyNetworkPolicy(ctx context.Context, dependencyName string, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	dependencyNetworkPolicy := generateDoguDepNetPol(doguResource, dogu, dependencyName)

	if err := u.upsertNetworkPolicy(ctx, dependencyNetworkPolicy); err != nil {
		return fmt.Errorf("failed to create or update network policy allow rule for dependency %s of dogu %s: %w", dependencyName, dogu.GetSimpleName(), err)
	}

	return nil
}

func (u *upserter) upsertComponentDependencyNetworkPolicy(ctx context.Context, dependencyName string, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	dependencyNetworkPolicy := generateComponentDepNetPol(doguResource, dogu, dependencyName)

	if err := u.upsertNetworkPolicy(ctx, dependencyNetworkPolicy); err != nil {
		return fmt.Errorf("failed to create or update network policy allow rule for dependency %s of dogu %s: %w", dependencyName, dogu.GetSimpleName(), err)
	}

	return nil
}

func (u *upserter) upsertNetworkPolicy(ctx context.Context, netPol *netv1.NetworkPolicy) error {
	return u.updateOrInsert(ctx, getNetPolObjectKey(netPol), &netv1.NetworkPolicy{}, netPol)
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
