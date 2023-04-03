package annotation

import (
	"fmt"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"strconv"
	"strings"
)

const (
	// CesServicesAnnotation contains the identifier of the annotation containing ces service information
	CesServicesAnnotation = "k8s-dogu-operator.cloudogu.com/ces-services"

	// serviceVarsPrefix defines the prefix for all service variables used to customize the service for the ecosystem
	serviceVarsPrefix = "SERVICE_"

	serviceAdditionalServices = "ADDITIONAL_SERVICES"
	serviceVarName            = "NAME"
	serviceVarLocation        = "LOCATION"
	serviceVarPass            = "PASS"
	serviceVarTags            = "TAGS"
	serviceTagWebapp          = "webapp"
)

// AdditionalIngressAnnotationsAnnotation contains additional ingress annotations to be appended to the ingress object for this service.
const AdditionalIngressAnnotationsAnnotation = "k8s-dogu-operator.cloudogu.com/additional-ingress-annotations"

// cesService describes a reachable service in the ecosystem.
type cesService struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Location string `json:"location"`
	Pass     string `json:"pass"`
}

// CesServiceAnnotator collects ces service information and annotates them to a given K8s service.
type CesServiceAnnotator struct{}

// AnnotateService annotates a given service with ces service information based on the given service and the provided
// image configuration which includes defined environment variables and labels used to customize the service for the
// ecosystem.
func (c *CesServiceAnnotator) AnnotateService(service *corev1.Service, config *imagev1.Config, additionalIngressAnnotations map[string]string) error {
	serviceTags, err := getServiceVariables(config)
	if err != nil {
		return fmt.Errorf("failed to get service tags: %w", err)
	}

	cesServices, err := createCesServices(service, config, serviceTags)
	if err != nil {
		return fmt.Errorf("failed to create ces services: %w", err)
	}

	err = appendServiceAnnotations(service, cesServices)
	if err != nil {
		return fmt.Errorf("failed to append annotation [%s] to service [%s]: %w", CesServicesAnnotation, service.GetName(), err)
	}

	err = appendAdditionalIngressAnnotations(service, additionalIngressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to append annotation [%s] to service [%s]: %w", AdditionalIngressAnnotationsAnnotation, service.GetName(), err)
	}

	return nil
}

func appendAdditionalIngressAnnotations(service *corev1.Service, ingressAnnotations map[string]string) error {
	if len(ingressAnnotations) < 1 {
		return nil
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	ingressAnnotationsJson, err := json.Marshal(ingressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to marshal additional ingress annotations: %w", err)
	}

	service.Annotations[AdditionalIngressAnnotationsAnnotation] = string(ingressAnnotationsJson)
	return nil
}

func appendServiceAnnotations(service *corev1.Service, cesServices []cesService) error {
	if len(cesServices) < 1 {
		return nil
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	cesServicesJson, err := json.Marshal(&cesServices)
	if err != nil {
		return fmt.Errorf("failed to marshal ces services: %w", err)
	}

	service.Annotations[CesServicesAnnotation] = string(cesServicesJson)
	return nil
}

func getServiceVariables(config *imagev1.Config) (map[string]string, error) {
	serviceVariables, err := getServiceVariablesFromEnvVariables(config.Env)
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to get service variables from environment variables: %w", err)
	}

	labelServiceVariables := getServiceVariablesFromLabels(config.Labels)
	for serviceVariable, value := range labelServiceVariables {
		serviceVariables[serviceVariable] = value
	}

	return serviceVariables, nil
}

func getServiceVariablesFromLabels(labels map[string]string) map[string]string {
	serviceVariables := map[string]string{}

	for label, value := range labels {
		if strings.HasPrefix(label, serviceVarsPrefix) {
			trimmedVariable := strings.TrimPrefix(label, serviceVarsPrefix)
			serviceVariables[trimmedVariable] = value
		}
	}

	return serviceVariables
}

func getServiceVariablesFromEnvVariables(envVariables []string) (map[string]string, error) {
	serviceVariables := map[string]string{}

	for _, envVariables := range envVariables {
		if strings.HasPrefix(envVariables, serviceVarsPrefix) {
			name, value, err := splitEnvVariable(envVariables)
			if err != nil {
				return map[string]string{}, fmt.Errorf("failed to split environment variable: %w", err)
			}

			trimmedVariable := strings.TrimPrefix(name, serviceVarsPrefix)
			serviceVariables[trimmedVariable] = value
		}
	}

	return serviceVariables, nil
}

func splitEnvVariable(variable string) (string, string, error) {
	variableParts := strings.SplitN(variable, "=", 2)

	if len(variableParts) != 2 {
		return "", "", fmt.Errorf("environment variable [%s] needs to be in form NAME=VALUE", variable)
	}

	return variableParts[0], variableParts[1], nil
}

func createCesServices(service *corev1.Service, config *imagev1.Config, serviceVariables map[string]string) ([]cesService, error) {
	var cesServices []cesService

	webAppServices, err := createWebAppCesServices(service, config, serviceVariables)
	if err != nil {
		return []cesService{}, fmt.Errorf("failed to create web app ces services: %w", err)
	}
	cesServices = append(cesServices, webAppServices...)

	defaultPort := getDefaultPortFromService(service)

	additionalServices, err := createAdditionalCesServices(serviceVariables, defaultPort)
	if err != nil {
		return []cesService{}, fmt.Errorf("failed to create additional ces services: %w", err)
	}
	cesServices = append(cesServices, additionalServices...)

	return cesServices, nil
}

func getDefaultPortFromService(service *corev1.Service) int32 {
	if len(service.Spec.Ports) == 0 {
		return 0
	}

	return service.Spec.Ports[0].Port
}

func createWebAppCesServices(service *corev1.Service, config *imagev1.Config, serviceVariables map[string]string) ([]cesService, error) {
	var webAppServices []cesService

	for exposedPort := range config.ExposedPorts {
		if webAppServices == nil {
			webAppServices = []cesService{}
		}

		port, protocol, err := SplitImagePortConfig(exposedPort)
		if err != nil {
			return []cesService{}, fmt.Errorf("error splitting port config: %w", err)
		}

		if !isServiceWebApp(port, protocol, serviceVariables) {
			continue
		}

		name := getServiceName(serviceVariables, port, service)
		location := getServiceLocation(serviceVariables, port, name)
		pass := getServicePass(serviceVariables, port, name)

		cesService := cesService{
			Name:     name,
			Port:     port,
			Location: location,
			Pass:     pass,
		}

		webAppServices = append(webAppServices, cesService)
	}

	return webAppServices, nil
}

func isServiceWebApp(port int32, protocol corev1.Protocol, serviceVariables map[string]string) bool {
	if protocol != corev1.ProtocolTCP {
		// can only create ces services for tcp ports
		return false
	}

	// service is webapp when SERVICE_TAGS contains `webapp`
	if hasTag(serviceVariables, serviceVarTags, serviceTagWebapp) {
		return true
	}

	// service is also webapp when SERVICE_<PORT>_TAGS contains `webapp`
	return hasTag(serviceVariables, fmt.Sprintf("%d_%s", port, serviceVarTags), serviceTagWebapp)
}

func hasTag(serviceVariables map[string]string, tagListName string, tag string) bool {
	tagList, hasTags := serviceVariables[tagListName]

	if hasTags {
		tags := strings.Split(tagList, ",")
		for _, t := range tags {
			if strings.ToLower(t) == tag {
				return true
			}
		}
	}

	return false
}

func getServiceName(serviceVariables map[string]string, port int32, service *corev1.Service) string {
	name := getValueFromServiceVariables(serviceVariables, port, serviceVarName)

	if name == "" {
		return service.GetName()
	}
	return name
}

func getServiceLocation(serviceVariables map[string]string, port int32, serviceName string) string {
	location := getValueFromServiceVariables(serviceVariables, port, serviceVarLocation)

	if location == "" {
		return fmt.Sprintf("/%s", serviceName)
	}

	// location should always contain a leading slash
	if !strings.HasPrefix(location, "/") {
		return fmt.Sprintf("/%s", location)
	}
	return location
}

func getServicePass(serviceVariables map[string]string, port int32, serviceName string) string {
	pass := getValueFromServiceVariables(serviceVariables, port, serviceVarPass)

	if pass == "" {
		return fmt.Sprintf("/%s", serviceName)
	}

	// pass should always contain a leading slash
	if !strings.HasPrefix(pass, "/") {
		return fmt.Sprintf("/%s", pass)
	}
	return pass
}

func getValueFromServiceVariables(serviceVariables map[string]string, port int32, variableName string) string {
	value := ""

	// values can be overwritten by SERVICE_NAME
	name, hasName := serviceVariables[variableName]
	if hasName {
		value = name
	}

	// values can be overwritten by SERVICE_<PORT>_NAME
	portName, hasPortName := serviceVariables[fmt.Sprintf("%d_%s", port, variableName)]
	if hasPortName {
		value = portName
	}

	return value
}

func createAdditionalCesServices(serviceVariables map[string]string, defaultPort int32) ([]cesService, error) {
	additionalCesServicesString, hasAdditionalServices := serviceVariables[serviceAdditionalServices]

	if hasAdditionalServices {
		var additionalCesServices []cesService
		err := json.Unmarshal([]byte(additionalCesServicesString), &additionalCesServices)
		if err != nil {
			return []cesService{}, fmt.Errorf("failed to unmarshal additional services: %w", err)
		}

		for i, service := range additionalCesServices {
			// location should always contain a leading slash
			if !strings.HasPrefix(service.Location, "/") {
				additionalCesServices[i].Location = fmt.Sprintf("/%s", service.Location)
			}

			// pass should always contain a leading slash
			if !strings.HasPrefix(service.Pass, "/") {
				additionalCesServices[i].Pass = fmt.Sprintf("/%s", service.Pass)
			}

			// port should by default be set to the default port
			if service.Port == 0 {
				additionalCesServices[i].Port = defaultPort
			}
		}

		return additionalCesServices, nil
	}

	return []cesService{}, nil
}

func SplitImagePortConfig(exposedPort string) (int32, corev1.Protocol, error) {
	portAndPotentiallyProtocol := strings.Split(exposedPort, "/")

	port, err := strconv.Atoi(portAndPotentiallyProtocol[0])
	if err != nil {
		return 0, "", fmt.Errorf("error parsing int: %w", err)
	}

	if len(portAndPotentiallyProtocol) == 2 {
		return int32(port), corev1.Protocol(strings.ToUpper(portAndPotentiallyProtocol[1])), nil
	}

	return int32(port), corev1.ProtocolTCP, nil
}
