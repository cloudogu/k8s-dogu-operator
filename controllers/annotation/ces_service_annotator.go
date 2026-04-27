package annotation

import (
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposition"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	// CesServicesAnnotation contains the identifier of the annotation containing ces service information
	CesServicesAnnotation = "k8s-dogu-operator.cloudogu.com/ces-services"
)

type cesService struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Location string `json:"location"`
	Pass     string `json:"pass"`
	Rewrite  string `json:"rewrite,omitempty"`
}

// CesServiceAnnotator collects ces service information and annotates them to a given K8s service.
type CesServiceAnnotator struct{}

// AnnotateService annotates a given service with ces service information based on the given service and the provided
// image configuration which includes defined environment variables and labels used to customize the service for the
// ecosystem.
func (c *CesServiceAnnotator) AnnotateService(service *corev1.Service, config *imagev1.Config) error {
	routes, err := exposition.CollectRoutes(service, config)
	if err != nil {
		return fmt.Errorf("failed to collect web routes: %w", err)
	}

	err = appendServiceAnnotations(service, routes)
	if err != nil {
		return fmt.Errorf("failed to append annotation [%s] to service [%s]: %w", CesServicesAnnotation, service.GetName(), err)
	}

	return nil
}

func appendServiceAnnotations(service *corev1.Service, routes []exposition.Route) error {
	if len(routes) < 1 {
		return nil
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	cesServices := make([]cesService, 0, len(routes))
	for _, route := range routes {
		cesServices = append(cesServices, cesService{
			Name:     route.Name,
			Port:     route.Port,
			Location: route.Path,
			Pass:     route.TargetPath,
			Rewrite:  route.Rewrite,
		})
	}

	cesServicesJSON, err := json.Marshal(&cesServices)
	if err != nil {
		return fmt.Errorf("failed to marshal ces services: %w", err)
	}

	service.Annotations[CesServicesAnnotation] = string(cesServicesJSON)
	return nil
}
