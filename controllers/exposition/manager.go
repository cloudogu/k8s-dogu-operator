package exposition

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	expClientV1 "github.com/cloudogu/k8s-exposition-lib/client/typed/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
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

func (em *ExpositionManager) EnsureExposition(ctx context.Context, doguResource *doguv2.Dogu) error {
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

	service := buildServiceForRoutes(doguResource, imageConfig)
	routes, err := CollectRoutes(service, &imageConfig.Config)
	if err != nil {
		return fmt.Errorf("failed to collect web routes: %w", err)
	}

	if len(routes) == 0 {
		return em.RemoveExposition(ctx, doguResource.GetSimpleDoguName())
	}

	desired := &expv1.Exposition{
		ObjectMeta: metav1.ObjectMeta{
			Name: createExpositionName(doguResource.Name),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind("Dogu")),
			},
		},
		Spec: BuildSpec(doguResource.Name, routes),
	}

	_, err = em.ensureExposition(ctx, desired)
	return err
}

func (em *ExpositionManager) RemoveExposition(ctx context.Context, doguName cescommons.SimpleName) error {
	err := em.client.Delete(ctx, createExpositionName(doguName.String()), metav1.DeleteOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete Exposition: %w", err)
	}

	return nil
}

func createExpositionName(doguName string) string {
	return fmt.Sprintf("%s-exposition", doguName)
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

func buildServiceForRoutes(doguResource *doguv2.Dogu, imageConfig *imagev1.ConfigFile) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: doguResource.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
		},
	}

	for exposedPort := range imageConfig.Config.ExposedPorts {
		port, protocol, err := SplitImagePortConfig(exposedPort)
		if err != nil {
			continue
		}

		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     strconv.Itoa(int(port)),
			Protocol: protocol,
			Port:     port,
		})
	}

	return service
}
