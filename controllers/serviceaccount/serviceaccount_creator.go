package serviceaccount

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/dogu"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// doguKind describes a service account on a dogu.
const (
	doguKind      = "dogu"
	cesKind       = "ces"
	componentKind = "component"
)

const (
	k8sCesControl = "k8s-ces-control"
)

// creator is the unit to handle the creation of service accounts
type creator struct {
	client                   client.Client
	sensitiveDoguCfgProvider SensitiveDoguConfigProvider
	doguFetcher              cloudogu.LocalDoguFetcher
	localDoguRegistry        dogu.LocalRegistry
	executor                 cloudogu.CommandExecutor
	clientSet                kubernetes.Interface
	apiClient                serviceAccountApiClient
	namespace                string
}

// NewCreator creates a new instance of ServiceAccountCreator
func NewCreator(provider SensitiveDoguConfigProvider, localDoguRegistry dogu.LocalRegistry, commandExecutor cloudogu.CommandExecutor, client client.Client, clientSet kubernetes.Interface, namespace string) *creator {
	localFetcher := cesregistry.NewLocalDoguFetcher(localDoguRegistry)
	return &creator{
		client:                   client,
		sensitiveDoguCfgProvider: provider,
		doguFetcher:              localFetcher,
		executor:                 commandExecutor,
		clientSet:                clientSet,
		apiClient:                &apiClient{},
		namespace:                namespace,
		localDoguRegistry:        localDoguRegistry,
	}
}

// CreateAll creates all service accounts for a given dogu. Existing service accounts will be skipped.
func (c *creator) CreateAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type

		senDoguCfg, err := c.sensitiveDoguCfgProvider.GetSensitiveDoguConfig(ctx, dogu.GetSimpleName())
		if err != nil {
			return fmt.Errorf("unable to get sensitive dogu config for dogu %s: %w", dogu.GetSimpleName(), err)
		}

		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			lErr := c.createDoguServiceAccount(ctx, dogu, senDoguCfg, serviceAccount, registryCredentialPath)
			if lErr != nil {
				return fmt.Errorf("unable to create service account for dogu %s: %w", serviceAccount.Type, lErr)
			}
		case componentKind:
			lErr := c.createComponentServiceAccount(ctx, dogu, senDoguCfg, serviceAccount, registryCredentialPath)
			if lErr != nil {
				return fmt.Errorf("unable to create service account for component %s: %w", serviceAccount.Type, lErr)
			}
		default:
			logger.Error(fmt.Errorf("unknown service account kind: %s", serviceAccount.Kind), "skipping service account creation")
			continue
		}
	}

	return nil
}

func (c *creator) createDoguServiceAccount(ctx context.Context, dogu *core.Dogu, senDoguCfg SensitiveDoguConfig,
	serviceAccount core.ServiceAccount, registryCredentialPath string) error {
	logger := log.FromContext(ctx)
	skip, err := serviceAccountExists(ctx, registryCredentialPath, senDoguCfg)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	enabled, err := c.localDoguRegistry.IsEnabled(ctx, serviceAccount.Type)
	if err != nil {
		return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
	}

	if !enabled && c.isOptionalServiceAccount(dogu, serviceAccount.Type) {
		logger.Info(fmt.Sprintf("skipping optional service account creation for %s, because the dogu is not installed", serviceAccount.Type))
		return nil
	}

	if !enabled && c.containsDependency(dogu.Dependencies, serviceAccount.Type) {
		return fmt.Errorf("service account dogu is not enabled and not optional")
	}

	err = c.create(ctx, dogu, serviceAccount, senDoguCfg)
	if err != nil {
		return err
	}

	return nil
}

func (c *creator) create(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, senDoguCfg SensitiveDoguConfigSetter) error {
	saDogu, err := c.doguFetcher.FetchInstalled(ctx, serviceAccount.Type)
	if err != nil {
		return fmt.Errorf("failed to get service account dogu.json: %w", err)
	}

	serviceAccountPod, err := getPodForServiceAccountDogu(ctx, c.client, saDogu)
	if err != nil {
		return fmt.Errorf("could not find service account producer pod %s: %w", saDogu.GetSimpleName(), err)
	}

	saCreds, err := c.executeCommand(ctx, dogu, saDogu, serviceAccountPod, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to execute service account create command: %w", err)
	}

	err = c.writeServiceAccounts(ctx, senDoguCfg, serviceAccount, saCreds)
	if err != nil {
		return fmt.Errorf("failed to write the service account credentials: %w", err)
	}
	return nil
}

func serviceAccountExists(ctx context.Context, registryCredentialPath string, senDoguCfg SensitiveDoguConfigGetter) (bool, error) {
	exists, err := senDoguCfg.Exists(ctx, registryCredentialPath)
	if err != nil {
		return false, fmt.Errorf("failed to check if service account already exists: %w", err)
	}
	if exists {
		return true, nil
	}

	return false, nil
}

func getPodForServiceAccountDogu(ctx context.Context, client client.Client, saDogu *core.Dogu) (*corev1.Pod, error) {
	versionlessDoguLabel := map[string]string{v1.DoguLabelName: saDogu.GetSimpleName()}
	return v1.GetPodForLabels(ctx, client, versionlessDoguLabel)
}

func (c *creator) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, saPod *corev1.Pod, serviceAccount core.ServiceAccount) (map[string]string, error) {
	createCommand, err := getExposedCommand(saDogu, "service-account-create")
	if err != nil {
		return nil, err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())

	command := exec.NewShellCommand(createCommand.Command, args...)
	buffer, err := c.executor.ExecCommandForPod(ctx, saPod, command, cloudogu.PodReady)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	saCreds, err := c.parseServiceCommandOutput(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account: %w", err)
	}

	return saCreds, nil
}

func (c *creator) writeServiceAccounts(ctx context.Context, senDoguCfg SensitiveDoguConfigSetter, serviceAccount core.ServiceAccount, saCreds map[string]string) error {
	for key, value := range saCreds {
		path := "/sa-" + serviceAccount.Type + "/" + key

		if err := senDoguCfg.Set(ctx, path, value); err != nil {
			return fmt.Errorf("failed to set sa value of key %s to sensisitive dogu config: %w", key, err)
		}
	}

	return nil
}

func getExposedCommand(dogu *core.Dogu, command string) (*core.ExposedCommand, error) {
	if !dogu.HasExposedCommand(command) {
		return nil, fmt.Errorf("service account dogu %s does not expose %s command", dogu.GetSimpleName(), command)
	}

	return dogu.GetExposedCommand(command), nil
}

func (c *creator) isOptionalServiceAccount(dogu *core.Dogu, sa string) bool {
	if c.containsDependency(dogu.Dependencies, sa) {
		return false
	} else if c.containsDependency(dogu.OptionalDependencies, sa) {
		return true
	}
	return false
}

func (c *creator) containsDependency(slice []core.Dependency, dependencyName string) bool {
	if slice == nil {
		return false
	}
	for _, s := range slice {
		if s.Name == dependencyName {
			return true
		}
	}
	return false
}

func (c *creator) parseServiceCommandOutput(output io.Reader) (map[string]string, error) {
	serviceAccountSettings := make(map[string]string)

	reader := bufio.NewReader(output)
	var line []byte
	var err error
	for err == nil {
		line, _, err = reader.ReadLine()
		if err != nil && err != io.EOF {
			return nil, err
		}

		// convert to string, trim and split by :
		lineAsString := strings.TrimSpace(string(line))
		if len(lineAsString) > 0 {
			kvs := strings.Split(lineAsString, ":")
			if len(kvs) != 2 {
				return nil, fmt.Errorf("invalid output from service account command on dogu")
			}

			serviceAccountSettings[strings.TrimSpace(kvs[0])] = strings.TrimSpace(kvs[1])
		}
	}

	return serviceAccountSettings, nil
}
