package exposition

import (
	"context"
	"fmt"
	"reflect"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	expClientV1 "github.com/cloudogu/k8s-exposition-lib/client/typed/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExpositionManager struct {
	client        expositionClient
	doguFetcher   localDoguFetcher
	imageRegistry imageRegistry
}

func NewManager(client expClientV1.ExpositionInterface, doguFetcher cesregistry.LocalDoguFetcher, imageRegistry imageregistry.ImageRegistry) *ExpositionManager {
	return &ExpositionManager{
		client:        client,
		doguFetcher:   doguFetcher,
		imageRegistry: imageRegistry,
	}
}

func (em *ExpositionManager) EnsureExposition(ctx context.Context, doguResource *doguv2.Dogu, doguService *corev1.Service) error {
	if doguResource == nil {
		return fmt.Errorf("dogu resource must not be nil")
	}

	doguDescriptor, err := em.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	imageConfig, err := em.imageRegistry.PullImageConfig(ctx, doguDescriptor.Image+":"+doguDescriptor.Version)
	if err != nil {
		return fmt.Errorf("failed to pull image config: %w", err)
	}

	routes, err := serviceaccess.CollectRoutes(doguService, &imageConfig.Config)
	if err != nil {
		return fmt.Errorf("failed to collect web routes: %w", err)
	}

	exposedPorts := serviceaccess.CollectExposedPorts(doguDescriptor)

	if len(routes) == 0 && len(exposedPorts) == 0 {
		return em.RemoveExposition(ctx, doguResource.GetSimpleDoguName())
	}

	spec, err := buildSpec(doguResource.Name, routes, exposedPorts)
	if err != nil {
		return fmt.Errorf("failed to build Exposition spec: %w", err)
	}

	desired := &expv1.Exposition{
		ObjectMeta: metav1.ObjectMeta{
			Name: doguResource.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind("Dogu")),
			},
		},
		Spec: spec,
	}

	_, err = em.ensureExposition(ctx, desired)
	return err
}

func (em *ExpositionManager) RemoveExposition(ctx context.Context, doguName cescommons.SimpleName) error {
	err := em.client.Delete(ctx, doguName.String(), metav1.DeleteOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete Exposition: %w", err)
	}

	return nil
}

func (em *ExpositionManager) ensureExposition(ctx context.Context, desired *expv1.Exposition) (*expv1.Exposition, error) {
	current, err := em.client.Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			created, createErr := em.client.Create(ctx, desired, metav1.CreateOptions{})
			if createErr != nil {
				return nil, fmt.Errorf("failed to create Exposition: %w", createErr)
			}
			return created, nil
		}

		return nil, fmt.Errorf("failed to get Exposition: %w", err)
	}

	if reflect.DeepEqual(current.Spec, desired.Spec) && reflect.DeepEqual(current.OwnerReferences, desired.OwnerReferences) {
		return current, nil
	}

	current.Spec = desired.Spec
	current.OwnerReferences = desired.OwnerReferences

	updated, err := em.client.Update(ctx, current, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update Exposition: %w", err)
	}

	return updated, nil
}

// buildSpec maps collected legacy routes and exposed ports to an Exposition spec.
func buildSpec(serviceName string, routes []serviceaccess.Route, exposedPorts []serviceaccess.ExposedPort) (expv1.ExpositionSpec, error) {
	httpEntries, err := buildHTTPEntries(serviceName, routes)
	if err != nil {
		return expv1.ExpositionSpec{}, err
	}

	return expv1.ExpositionSpec{
		HTTP: httpEntries,
		TCP:  buildTCPEntries(serviceName, exposedPorts),
		UDP:  buildUDPEntries(serviceName, exposedPorts),
	}, nil
}
