package config

import (
	"fmt"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

var (
	envVarNamespace              = "NAMESPACE"
	envVarDoguRegistryEndpoint   = "DOGU_REGISTRY_ENDPOINT"
	envVarDoguRegistryUsername   = "DOGU_REGISTRY_USERNAME"
	envVarDoguRegistryPassword   = "DOGU_REGISTRY_PASSWORD"
	envVarDockerRegistryUsername = "DOCKER_REGISTRY_USERNAME"
	envVarDockerRegistryPassword = "DOCKER_REGISTRY_PASSWORD"
	// logModeEnvVar is the constant for env variable ZAP_DEVELOPMENT_MODE
	// which specifies the development mode for zap options. Valid values are
	// true or false. In development mode the logger produces stacktraces on warnings and no smapling.
	// In regular mode (default) the logger produces stacktraces on errors and sampling
	envVarLogMode = "ZAP_DEVELOPMENT_MODE"
	log           = ctrl.Log.WithName("config")
)

// DoguRegistryData contains all necessary data for the dogu registry.
type DoguRegistryData struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// DockerRegistryData contains all necessary data for the Docker registry.
type DockerRegistryData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// OperatorConfig contains all configurable values for the dogu operator.
type OperatorConfig struct {
	// Namespace specifies the namespace that the operator is deployed to.
	Namespace string `json:"namespace"`
	// DoguRegistry contains all necessary data for the dogu registry.
	DoguRegistry DoguRegistryData `json:"dogu_registry"`
	// DockerRegistry contains all necessary data for the Docker registry.
	DockerRegistry DockerRegistryData `json:"docker_registry"`
	// DevelopmentLogMode determines whether the development mode should be used when logging
	DevelopmentLogMode bool `json:"development_log_mode"`
}

func NewOperatorConfig() (*OperatorConfig, error) {
	namespace, err := readNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}
	log.Info(fmt.Sprintf("Deploying the k8s dogu operator in namespace %s", namespace))

	logLevel, err := readZapLogLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}

	doguRegistryData, err := readDoguRegistryData()
	if err != nil {
		return nil, fmt.Errorf("failed to read dogu registry data: %w", err)
	}
	log.Info(fmt.Sprintf("Found stored dogu registry data! Using dogu registry %s", doguRegistryData.Endpoint))

	dockerRegistryData, err := readDockerRegistryData()
	if err != nil {
		return nil, fmt.Errorf("failed to read dogu registry data: %w", err)
	}
	log.Info("Found stored docker registry data!")

	return &OperatorConfig{
		Namespace:          namespace,
		DoguRegistry:       doguRegistryData,
		DockerRegistry:     dockerRegistryData,
		DevelopmentLogMode: logLevel,
	}, nil
}

func readNamespace() (string, error) {
	namespace, err := getEnvVar(envVarNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", envVarNamespace, err)
	}

	return namespace, nil
}

func readZapLogLevel() (bool, error) {
	logMode := false
	logModeEnv, err := getEnvVar(envVarLogMode)

	if err == nil {
		logMode, err = strconv.ParseBool(logModeEnv)
		if err != nil {
			return false, fmt.Errorf("failed to parse %s; valid values are true or false: %w", logModeEnv, err)
		}
	}

	return logMode, nil
}

func readDoguRegistryData() (DoguRegistryData, error) {
	endpoint, err := getEnvVar(envVarDoguRegistryEndpoint)
	if err != nil {
		return DoguRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDoguRegistryEndpoint, err)
	}
	// remove tailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	username, err := getEnvVar(envVarDoguRegistryUsername)
	if err != nil {
		return DoguRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDoguRegistryUsername, err)
	}

	password, err := getEnvVar(envVarDoguRegistryPassword)
	if err != nil {
		return DoguRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDoguRegistryPassword, err)
	}

	return DoguRegistryData{
		Endpoint: endpoint,
		Username: username,
		Password: password,
	}, nil
}

func readDockerRegistryData() (DockerRegistryData, error) {
	username, err := getEnvVar(envVarDockerRegistryUsername)
	if err != nil {
		return DockerRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDockerRegistryUsername, err)
	}

	password, err := getEnvVar(envVarDockerRegistryPassword)
	if err != nil {
		return DockerRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDockerRegistryPassword, err)
	}

	return DockerRegistryData{
		Username: username,
		Password: password,
	}, nil
}

func getEnvVar(name string) (string, error) {
	ns, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return ns, nil
}
