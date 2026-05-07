package serviceaccess

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	serviceVarsPrefix = "SERVICE_"

	serviceAdditionalServices = "ADDITIONAL_SERVICES"
	serviceVarName            = "NAME"
	serviceVarLocation        = "LOCATION"
	serviceVarPass            = "PASS"
	serviceVarRewrite         = "REWRITE"
	serviceVarTags            = "TAGS"
	serviceTagWebapp          = "webapp"
)

// CollectRoutes extracts legacy v2 web route data from service and image config.
func CollectRoutes(service *corev1.Service, config *imagev1.Config) ([]Route, error) {
	serviceVariables, err := getServiceVariables(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get service tags: %w", err)
	}

	routes, err := createRoutes(service, config, serviceVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to create routes: %w", err)
	}

	return routes, nil
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

func createRoutes(service *corev1.Service, config *imagev1.Config, serviceVariables map[string]string) ([]Route, error) {
	var routes []Route

	webAppRoutes, err := createWebAppRoutes(service, config, serviceVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to create web app routes: %w", err)
	}
	routes = append(routes, webAppRoutes...)

	defaultPort := getDefaultPortFromService(service)

	additionalRoutes, err := createAdditionalRoutes(serviceVariables, defaultPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create additional routes: %w", err)
	}
	routes = append(routes, additionalRoutes...)

	return routes, nil
}

func getDefaultPortFromService(service *corev1.Service) int32 {
	if len(service.Spec.Ports) == 0 {
		return 0
	}

	return service.Spec.Ports[0].Port
}

func createWebAppRoutes(service *corev1.Service, config *imagev1.Config, serviceVariables map[string]string) ([]Route, error) {
	var webAppRoutes []Route

	for exposedPort := range config.ExposedPorts {
		port, protocol, err := SplitImagePortConfig(exposedPort)
		if err != nil {
			return nil, fmt.Errorf("error splitting port config: %w", err)
		}

		if !isServiceWebApp(port, protocol, serviceVariables) {
			continue
		}

		name := getServiceName(serviceVariables, port, service)
		location := getServicePath(serviceVariables, port, name)
		pass := getServiceTargetPath(serviceVariables, port, name)
		serviceRewrite := getValueFromServiceVariables(serviceVariables, port, serviceVarRewrite)

		webAppRoutes = append(webAppRoutes, Route{
			Name:     name,
			Port:     port,
			Location: location,
			Pass:     pass,
			Rewrite:  serviceRewrite,
		})
	}

	return webAppRoutes, nil
}

func isServiceWebApp(port int32, protocol corev1.Protocol, serviceVariables map[string]string) bool {
	if protocol != corev1.ProtocolTCP {
		return false
	}

	// Port-specific webapp tags must take precedence over the global tag.
	// Some dogus expose multiple TCP ports but only one of them serves HTTP.
	if hasAnyPortSpecificWebappTag(serviceVariables) {
		return hasWebappTag(serviceVariables, fmt.Sprintf("%d_%s", port, serviceVarTags))
	}

	if hasWebappTag(serviceVariables, serviceVarTags) {
		return true
	}

	return hasWebappTag(serviceVariables, fmt.Sprintf("%d_%s", port, serviceVarTags))
}

func hasAnyPortSpecificWebappTag(serviceVariables map[string]string) bool {
	for name := range serviceVariables {
		if !strings.HasSuffix(name, "_"+serviceVarTags) {
			continue
		}

		if _, err := strconv.Atoi(strings.TrimSuffix(name, "_"+serviceVarTags)); err != nil {
			continue
		}

		if hasWebappTag(serviceVariables, name) {
			return true
		}
	}

	return false
}

func hasWebappTag(serviceVariables map[string]string, tagListName string) bool {
	tagList, hasTags := serviceVariables[tagListName]

	if hasTags {
		tags := strings.Split(tagList, ",")
		for _, t := range tags {
			if strings.ToLower(t) == serviceTagWebapp {
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

func getServicePath(serviceVariables map[string]string, port int32, serviceName string) string {
	path := getValueFromServiceVariables(serviceVariables, port, serviceVarLocation)

	if path == "" {
		return fmt.Sprintf("/%s", serviceName)
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Sprintf("/%s", path)
	}
	return path
}

func getServiceTargetPath(serviceVariables map[string]string, port int32, serviceName string) string {
	targetPath := getValueFromServiceVariables(serviceVariables, port, serviceVarPass)

	if targetPath == "" {
		return fmt.Sprintf("/%s", serviceName)
	}

	if !strings.HasPrefix(targetPath, "/") {
		return fmt.Sprintf("/%s", targetPath)
	}
	return targetPath
}

func getValueFromServiceVariables(serviceVariables map[string]string, port int32, variableName string) string {
	value := ""

	name, hasName := serviceVariables[variableName]
	if hasName {
		value = name
	}

	portName, hasPortName := serviceVariables[fmt.Sprintf("%d_%s", port, variableName)]
	if hasPortName {
		value = portName
	}

	return value
}

func createAdditionalRoutes(serviceVariables map[string]string, defaultPort int32) ([]Route, error) {
	additionalRoutesString, hasAdditionalServices := serviceVariables[serviceAdditionalServices]

	if hasAdditionalServices {
		var additionalRoutes []Route
		err := json.Unmarshal([]byte(additionalRoutesString), &additionalRoutes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal additional services: %w", err)
		}

		for i, route := range additionalRoutes {
			if !strings.HasPrefix(route.Location, "/") {
				additionalRoutes[i].Location = fmt.Sprintf("/%s", route.Location)
			}

			if !strings.HasPrefix(route.Pass, "/") {
				additionalRoutes[i].Pass = fmt.Sprintf("/%s", route.Pass)
			}

			if route.Port == 0 {
				additionalRoutes[i].Port = defaultPort
			}
		}

		return additionalRoutes, nil
	}

	return nil, nil
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
