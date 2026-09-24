package config

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	dccv3 "github.com/cloudogu/dogu-lib/doguv3/doguregistry/dcc/config"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	StageDevelopment         = "development"
	StageProduction          = "production"
	StageEnvironmentVariable = "STAGE"
)

const (
	defaultRequeueTime                = time.Second * 5
	defaultHelmReconciliationInterval = time.Second * 15
)

const cacheDir = "/tmp/dogu-registry-cache"

const (
	// OperatorAdditionalImagesConfigmapName contains the configmap name which consists of auxiliary yet necessary container images.
	OperatorAdditionalImagesConfigmapName = "k8s-dogu-operator-additional-images"
	// ChownInitImageConfigmapNameKey contains the key to retrieve the chown init container image from the OperatorAdditionalImagesConfigmapName configmap.
	ChownInitImageConfigmapNameKey = "chownInitImage"
	// ExporterImageConfigmapNameKey contains the key to retrieve the image used for exporter-sidecar-container
	ExporterImageConfigmapNameKey = "exporterImage"
	// AdditionalMountsInitContainerImageConfigmapNameKey contains the key to retrieve the image used for the dogu-additional-mount-init-container
	AdditionalMountsInitContainerImageConfigmapNameKey = "additionalMountsInitContainerImage"
)

var Stage = StageProduction
var log = ctrl.Log.WithName("config")

const (
	envVarProxyUrl                                = "PROXY_URL"
	envVarNamespace                               = "NAMESPACE"
	envVarDoguV3RegistryEndpoint                  = "DOGU_V3_REGISTRY_ENDPOINT"
	envVarDoguV3RegistryUsername                  = "DOGU_V3_REGISTRY_USERNAME"
	envVarDoguV3RegistryPassword                  = "DOGU_V3_REGISTRY_PASSWORD"
	envVarDoguV3RegistryURLSchema                 = "DOGU_V3_REGISTRY_URLSCHEMA"
	envVarDoguV3RegistryInsecureSkipVerify        = "DOGU_V3_REGISTRY_INSECURE_SKIP_VERIFY"
	envVarDoguRegistryEndpoint                    = "DOGU_REGISTRY_ENDPOINT"
	envVarDoguRegistryUsername                    = "DOGU_REGISTRY_USERNAME"
	envVarDoguRegistryPassword                    = "DOGU_REGISTRY_PASSWORD"
	envVarDoguRegistryURLSchema                   = "DOGU_REGISTRY_URLSCHEMA"
	envVarNetworkPolicyEnabled                    = "NETWORK_POLICIES_ENABLED"
	envVarAuthRegistrationEnabled                 = "AUTH_REGISTRATION_ENABLED"
	envVarExpositionEnabled                       = "EXPOSITION_ENABLED"
	envVarWarpMenuEntryEnabled                    = "WARP_MENU_ENTRY_ENABLED"
	envVarDisablePostfixDependencyCheck           = "DISABLE_POSTFIX_DEPENDENCY_CHECK"
	envVarRequeueTimeForDoguResourceInNanoseconds = "REQUEUE_TIME_FOR_DOGU_RESOURCE_IN_NANOSECONDS"
	envVarImageConfigCacheSize                    = "IMAGE_CONFIG_CACHE_SIZE"
	envVarHelmReconciliationInterval              = "HELM_RECONCILIATION_INTERVAL"
	errMsgFailedToParseEnvVarValue                = "failed to parse value of environment variable %s: %w"
)

// defaultImageConfigCacheSize is the number of image configs kept in the in-memory cache when
// IMAGE_CONFIG_CACHE_SIZE is unset or invalid.
const defaultImageConfigCacheSize = 50

const v3DoguRegistryMissingAttributeFmt = "Dogu V3 registry %s not set. Using v3 dogus requires to set the attribute in the secret dogu-registry-v3"

// DoguRegistryData contains all necessary data for the dogu registry.
type DoguRegistryData struct {
	Endpoint  string `json:"endpoint"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	URLSchema string `json:"urlschema"`
}

// OperatorConfig contains all configurable values for the dogu operator.
type OperatorConfig struct {
	// Namespace specifies the namespace that the operator is deployed to.
	Namespace string `json:"namespace"`
	// DoguRegistry contains all necessary data for the dogu registry.
	DoguRegistry DoguRegistryData `json:"dogu_registry"`
	// DoguV3Registry contains all necessary data for the dogu v3 registry.
	DoguV3Registry DoguRegistryData `json:"dogu_v3_registry"`
	// Version contains the current version of the operator
	Version *core.Version `json:"version"`
	// NetworkPoliciesEnabled defines whether network policies should be created for dogus and their dependencies
	NetworkPoliciesEnabled bool `json:"network_policies_enabled"`
	// AuthRegistrationEnabled defines whether the operator should manage AuthRegistration CRs for v2 dogus.
	AuthRegistrationEnabled bool `json:"auth_registration_enabled"`
	// ExpositionEnabled defines whether the operator should manage Exposition CRs for v2 dogus.
	ExpositionEnabled bool `json:"exposition_enabled"`
	// WarpMenuEntryEnabled defines whether the operator should manage WarpMenuEntry CRs for v2 dogus.
	WarpMenuEntryEnabled bool `json:"warpmenuentry_enabled"`
	// DisablePostfixDependencyCheck defines whether the operator should validate dependencies on postfix.
	// If set to false, the operator will assume that postfix is installed as a normal dogu and will validate the dependencies accordingly.
	// If set to true, the operator will assume that postfix is installed as a component and will not validate the dependencies.
	DisablePostfixDependencyCheck bool `json:"disable_postfix_dependency_check"`
	// RequeueTimeForDoguReconciler defines the requeue time for the dogu reconciler
	RequeueTimeForDoguReconciler time.Duration `json:"requeue_time_for_dogu_reconciler"`
	// ImageConfigCacheSize defines the maximum number of dogu image configs kept in the in-memory image config cache.
	// A value <= 0 disables the cache.
	ImageConfigCacheSize int `json:"image_config_cache_size"`
	// HelmReconciliationInterval defines the interval in which dogu Helm releases should be reconciled by helm-controller
	HelmReconciliationInterval time.Duration `json:"helm_reconciliation_interval"`
}

type Version string

// NewOperatorConfig creates a new operator config by reading values from the environment variables
func NewOperatorConfig(version Version) (*OperatorConfig, error) {
	Stage = getEnvVarWithDefault(StageEnvironmentVariable, Stage)
	if Stage == StageDevelopment {
		log.Info("Starting in development mode! This is not recommended for production!")
	}

	parsedVersion, err := core.ParseVersion(string(version))
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	log.Info(fmt.Sprintf("Version: [%s]", version))

	namespace, err := readNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}
	log.Info(fmt.Sprintf("Deploying the k8s dogu operator in namespace %s", namespace))

	doguRegistryData, err := readDoguRegistryData()
	if err != nil {
		return nil, fmt.Errorf("failed to read dogu registry data: %w", err)
	}
	log.Info(fmt.Sprintf("Found stored dogu registry data! Using dogu registry %s", doguRegistryData.Endpoint))

	doguReconcilerRequeueTime, err := readDoguReconcilerRequeueTime()
	if err != nil {
		return nil, fmt.Errorf("failed to read dogu reconciler requeue time: %w", err)
	}
	log.Info(fmt.Sprintf("Found stored dogu reconciler requeue time! Using requeue time %s", doguReconcilerRequeueTime.String()))

	helmReconciliationInterval, err := readHelmReconciliationInterval()
	if err != nil {
		return nil, fmt.Errorf("failed to read helm reconciliation interval: %w", err)
	}
	log.Info(fmt.Sprintf("Found stored helm reconciliation interval! Using interval %s", helmReconciliationInterval))

	return &OperatorConfig{
		Namespace:                     namespace,
		DoguRegistry:                  doguRegistryData,
		Version:                       &parsedVersion,
		NetworkPoliciesEnabled:        getNetworkPoliciesEnabled(),
		AuthRegistrationEnabled:       getAuthRegistrationEnabled(),
		ExpositionEnabled:             getExpositionEnabled(),
		WarpMenuEntryEnabled:          getWarpMenuEntryEnabled(),
		DisablePostfixDependencyCheck: getDisablePostfixDependencyCheck(),
		RequeueTimeForDoguReconciler:  doguReconcilerRequeueTime,
		ImageConfigCacheSize:          getImageConfigCacheSize(),
		DoguV3Registry:                readDoguV3RegistryData(),
		HelmReconciliationInterval:    helmReconciliationInterval,
	}, nil
}

func readNamespace() (string, error) {
	namespace, err := getRequiredEnvVar(envVarNamespace)
	if err != nil {
		return "", err
	}

	return namespace, nil
}

func readDoguReconcilerRequeueTime() (time.Duration, error) {
	requeueTimeString, err := getRequiredEnvVar(envVarRequeueTimeForDoguResourceInNanoseconds)
	if err != nil {
		return defaultRequeueTime, err
	}
	requeueTime, err := strconv.ParseFloat(requeueTimeString, 64)
	if err != nil {
		return defaultRequeueTime, err
	}
	return time.Duration(requeueTime), nil
}

// For now the dogu v3 registry is optional, so we do not return errors here if env vars are not found.
func readDoguV3RegistryData() DoguRegistryData {
	endpoint, found := os.LookupEnv(envVarDoguV3RegistryEndpoint)
	if !found {
		log.Info(fmt.Sprintf(v3DoguRegistryMissingAttributeFmt, "endpoint"))
	} else {
		// remove tailing slash
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	username, found := os.LookupEnv(envVarDoguV3RegistryUsername)
	if !found {
		log.Info(fmt.Sprintf(v3DoguRegistryMissingAttributeFmt, "username"))
	}

	password, found := os.LookupEnv(envVarDoguV3RegistryPassword)
	if !found {
		log.Info(fmt.Sprintf(v3DoguRegistryMissingAttributeFmt, "password"))
	}

	urlSchema, found := os.LookupEnv(envVarDoguV3RegistryURLSchema)
	if !found {
		urlSchema = "default"
	}

	return DoguRegistryData{
		Endpoint:  endpoint,
		Username:  username,
		Password:  password,
		URLSchema: urlSchema,
	}
}

func readDoguRegistryData() (DoguRegistryData, error) {
	endpoint, err := getRequiredEnvVar(envVarDoguRegistryEndpoint)
	if err != nil {
		return DoguRegistryData{}, err
	}
	// remove tailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	username, err := getRequiredEnvVar(envVarDoguRegistryUsername)
	if err != nil {
		return DoguRegistryData{}, err
	}

	password, err := getRequiredEnvVar(envVarDoguRegistryPassword)
	if err != nil {
		return DoguRegistryData{}, err
	}

	urlschema := getEnvVarWithDefault(envVarDoguRegistryURLSchema, "default")

	return DoguRegistryData{
		Endpoint:  endpoint,
		Username:  username,
		Password:  password,
		URLSchema: urlschema,
	}, nil
}

func getEnvVarWithDefault(name string, defaultValue string) string {
	value, found := os.LookupEnv(name)
	if !found {
		log.Info("environment variable not set, using default.", "variable", name, "default", defaultValue)
		return defaultValue
	}
	return value

}

func getRequiredEnvVar(name string) (string, error) {
	ns, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("failed to get env var [%s]: environment variable %s must be set", name, name)
	}
	return ns, nil
}

// GetRemoteConfiguration creates a remote configuration with the configured values.
func (o *OperatorConfig) GetRemoteConfiguration() (*core.Remote, error) {
	urlSchema := o.DoguRegistry.URLSchema
	if urlSchema != "index" {
		log.Info("URLSchema is not index. Setting it to default.")
		urlSchema = "default"
	}

	endpoint := o.DoguRegistry.Endpoint
	if urlSchema == "default" {
		// trim suffix 'dogus' or 'dogus/' to provide maximum compatibility with the old remote configuration of the operator
		endpoint = strings.TrimSuffix(endpoint, "dogus/")
		endpoint = strings.TrimSuffix(endpoint, "dogus")
	}

	proxySettings, err := getV2ProxySettings()
	if err != nil {
		return nil, err
	}

	return &core.Remote{
		Endpoint:      endpoint,
		CacheDir:      cacheDir,
		URLSchema:     urlSchema,
		ProxySettings: proxySettings,
	}, nil
}

// GetV3RemoteConfiguration creates a remote configuration with the configured values.
// It uses the default values for caching configuration from the dogu-lib.
func (o *OperatorConfig) GetV3RemoteConfiguration() (*dccv3.DoguRegistryConfiguration, error) {
	proxySettings, err := getV3ProxySettings()
	if err != nil {
		return nil, err
	}

	insecure := false
	env, b := os.LookupEnv(envVarDoguV3RegistryInsecureSkipVerify)
	if b && strings.ToLower(env) == "true" {
		log.Info("Dogu V3 registry insecure skip verify is set to true")
		insecure = true
	}

	return &dccv3.DoguRegistryConfiguration{
		BaseURL:            o.DoguV3Registry.Endpoint,
		ProxySettings:      proxySettings,
		InsecureSkipVerify: insecure,
		UserAgent:          fmt.Sprintf("k8s-dogu-operator/%s (%s/%s)", o.Version, runtime.GOOS, runtime.GOARCH),
		URLSchema:          o.DoguV3Registry.URLSchema,
	}, nil
}

func getV2ProxySettings() (core.ProxySettings, error) {
	settings, err := getV3ProxySettings()
	if err != nil {
		return core.ProxySettings{}, err
	}

	return core.ProxySettings{
		Enabled:  settings.Enabled,
		Server:   settings.Server,
		Port:     settings.Port,
		Username: settings.Username,
		Password: settings.Password,
	}, nil
}

func getV3ProxySettings() (dccv3.ProxySettings, error) {
	proxyURL, found := os.LookupEnv(envVarProxyUrl)
	proxySettings := dccv3.ProxySettings{}
	if found && len(proxyURL) > 0 {
		var err error
		if proxySettings, err = configureProxySettings(proxyURL); err != nil {
			return dccv3.ProxySettings{}, err
		}
	}

	return proxySettings, nil
}

func configureProxySettings(proxyURL string) (dccv3.ProxySettings, error) {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return dccv3.ProxySettings{}, fmt.Errorf("invalid proxy url: %w", err)
	}

	proxySettings := dccv3.ProxySettings{}
	proxySettings.Enabled = true
	if parsedURL.User != nil {
		proxySettings.Username = parsedURL.User.Username()
		if password, set := parsedURL.User.Password(); set {
			proxySettings.Password = password
		}
	}

	proxySettings.Server = parsedURL.Hostname()

	port, err := strconv.Atoi(parsedURL.Port())
	if err != nil {
		return dccv3.ProxySettings{}, fmt.Errorf("invalid port %s: %w", parsedURL.Port(), err)
	}
	proxySettings.Port = port

	return proxySettings, nil
}

// GetV3RemoteCredentials creates a remote credential pair for dogu v3 with the configured values.
func (o *OperatorConfig) GetV3RemoteCredentials() *dccv3.Credentials {
	return &dccv3.Credentials{
		Username: o.DoguV3Registry.Username,
		Password: o.DoguV3Registry.Password,
	}
}

// GetRemoteCredentials creates a remote credential pair with the configured values.
func (o *OperatorConfig) GetRemoteCredentials() *core.Credentials {
	return &core.Credentials{
		Username: o.DoguRegistry.Username,
		Password: o.DoguRegistry.Password,
	}
}

func getNetworkPoliciesEnabled() bool {
	netPolEnabledStr, found := os.LookupEnv(envVarNetworkPolicyEnabled)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Enabling network policies by default", envVarNetworkPolicyEnabled))
		return false
	}

	netPolEnabled, err := strconv.ParseBool(netPolEnabledStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarNetworkPolicyEnabled, err), "Enabling network policies by default")
		return true
	}

	return netPolEnabled
}

func getAuthRegistrationEnabled() bool {
	authRegistrationEnabledStr, found := os.LookupEnv(envVarAuthRegistrationEnabled)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Disabling auth registration by default", envVarAuthRegistrationEnabled))
		return false
	}

	authRegistrationEnabled, err := strconv.ParseBool(authRegistrationEnabledStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarAuthRegistrationEnabled, err), "Disabling auth registration by default")
		return false
	}

	return authRegistrationEnabled
}

func getExpositionEnabled() bool {
	expositionEnabledStr, found := os.LookupEnv(envVarExpositionEnabled)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Disabling exposition by default", envVarExpositionEnabled))
		return false
	}

	expositionEnabled, err := strconv.ParseBool(expositionEnabledStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarExpositionEnabled, err), "Disabling exposition by default")
		return false
	}

	return expositionEnabled
}

func getWarpMenuEntryEnabled() bool {
	warpMenuEntryEnabledStr, found := os.LookupEnv(envVarWarpMenuEntryEnabled)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Disabling warpMenuEntry by default", warpMenuEntryEnabledStr))
		return false
	}

	warpMenuEntryEnabled, err := strconv.ParseBool(warpMenuEntryEnabledStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarWarpMenuEntryEnabled, err), "Disabling warpMenuEntry by default")
		return false
	}

	return warpMenuEntryEnabled
}

func getDisablePostfixDependencyCheck() bool {
	disablePostfixDependencyCheckStr, found := os.LookupEnv(envVarDisablePostfixDependencyCheck)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Leaving postfix dependency check enabled", envVarDisablePostfixDependencyCheck))
		return false
	}

	disablePostfixDependencyCheck, err := strconv.ParseBool(disablePostfixDependencyCheckStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarDisablePostfixDependencyCheck, err), "Leaving postfix dependency check enabled")
		return false
	}

	return disablePostfixDependencyCheck
}

// getImageConfigCacheSize returns the maximum number of image configs kept in the in-memory image config cache.
// A value <= 0 disables the cache. It defaults to defaultImageConfigCacheSize when the environment variable is
// unset or cannot be parsed.
func getImageConfigCacheSize() int {
	sizeStr, found := os.LookupEnv(envVarImageConfigCacheSize)
	if !found {
		log.Info(fmt.Sprintf("Environment variable %s not set. Using default image config cache size %d", envVarImageConfigCacheSize, defaultImageConfigCacheSize))
		return defaultImageConfigCacheSize
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Error(fmt.Errorf(errMsgFailedToParseEnvVarValue, envVarImageConfigCacheSize, err), fmt.Sprintf("Using default image config cache size %d", defaultImageConfigCacheSize))
		return defaultImageConfigCacheSize
	}

	return size
}

func readHelmReconciliationInterval() (time.Duration, error) {
	timeString, err := getRequiredEnvVar(envVarHelmReconciliationInterval)
	if err != nil {
		return defaultHelmReconciliationInterval, err
	}
	duration, err := time.ParseDuration(timeString)
	if err != nil {
		return defaultHelmReconciliationInterval, fmt.Errorf("value of env var [%s] is no valid duration: %w", envVarHelmReconciliationInterval, err)
	}
	return duration, nil
}

func GetStage() (string, error) {
	stage, err := getRequiredEnvVar(StageEnvironmentVariable)
	if err != nil {
		return "", err
	}
	return stage, nil
}
