package config

import (
	"encoding/json"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

const (
	StageDevelopment         = "development"
	StageProduction          = "production"
	StageEnvironmentVariable = "STAGE"
	CacheDirProduction       = "/home/nonroot"
	CacheDirDevelopment      = "."
)

var Stage = StageProduction

var (
	envVarNamespace            = "NAMESPACE"
	envVarDoguRegistryEndpoint = "DOGU_REGISTRY_ENDPOINT"
	envVarDoguRegistryUsername = "DOGU_REGISTRY_USERNAME"
	envVarDoguRegistryPassword = "DOGU_REGISTRY_PASSWORD"
	envVarDockerRegistry       = "DOCKER_REGISTRY"
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

// DockerRegistrySecretData contains all registry login information from a Docker-JSON-config file.
type DockerRegistrySecretData struct {
	Auths map[string]DockerRegistryData `json:"auths"`
}

// DockerRegistryData contains all necessary data for the Docker registry.
type DockerRegistryData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
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
	// Version contains the current version of the operator
	Version *core.Version `json:"version"`
}

// NewOperatorConfig creates a new operator config by reading values from the environment variables
func NewOperatorConfig(version string) (*OperatorConfig, error) {
	stage, err := getEnvVar(StageEnvironmentVariable)
	if err != nil {
		log.Error(err, "Error reading stage environment variable. Use Stage production")
	}
	Stage = stage

	if Stage == StageDevelopment {
		log.Info("Starting in development mode! This is not recommended for production!")
	}

	parsedVersion, err := core.ParseVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	log.Info(fmt.Sprintf("Version: [%s]", version))

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
		Version:            &parsedVersion,
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
	dockerRegistryData, err := getEnvVar(envVarDockerRegistry)
	if err != nil {
		return DockerRegistryData{}, fmt.Errorf("failed to get env var [%s]: %w", envVarDockerRegistry, err)
	}

	var secretData DockerRegistrySecretData
	err = json.Unmarshal([]byte(dockerRegistryData), &secretData)
	if err != nil {
		return DockerRegistryData{}, fmt.Errorf("failed to unmarshal docker secret data: %w", err)
	}

	for _, data := range secretData.Auths {
		return data, nil
	}
	return DockerRegistryData{}, fmt.Errorf("no docker regsitry data provided")
}

func getEnvVar(name string) (string, error) {
	ns, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return ns, nil
}

// GetRemoteConfiguration creates a remote configuration with the configured values.
func (o *OperatorConfig) GetRemoteConfiguration() *core.Remote {
	endpoint := o.DoguRegistry.Endpoint
	// trim suffix 'dogus' or 'dogus/' to provide maximum compatibility with the old remote configuration of the operator
	endpoint = strings.TrimSuffix(endpoint, "dogus/")
	endpoint = strings.TrimSuffix(endpoint, "dogus")

	var cacheDir string

	if Stage == StageProduction {
		cacheDir = CacheDirProduction
	} else {
		cacheDir = CacheDirDevelopment
	}

	return &core.Remote{
		Endpoint: endpoint,
		CacheDir: cacheDir,
	}
}

// GetRemoteCredentials creates a remote credential pair with the configured values.
func (o *OperatorConfig) GetRemoteCredentials() *core.Credentials {
	return &core.Credentials{
		Username: o.DoguRegistry.Username,
		Password: o.DoguRegistry.Password,
	}
}
