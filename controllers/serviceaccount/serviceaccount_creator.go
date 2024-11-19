package serviceaccount

import (
	"bufio"
	"context"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	regLibErr "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
)

// doguKind describes a service account on a dogu.
const (
	doguKind      = "dogu"
	componentKind = "component"
)

// creator is the unit to handle the creation of service accounts
type creator struct {
	client            client.Client
	sensitiveDoguRepo sensitiveDoguConfigRepository
	doguFetcher       localDoguFetcher
	executor          exec.CommandExecutor
	clientSet         kubernetes.Interface
	apiClient         serviceAccountApiClient
	namespace         string
}

// NewCreator creates a new instance of ServiceAccountCreator
func NewCreator(repo sensitiveDoguConfigRepository, localDoguFetcher localDoguFetcher, commandExecutor exec.CommandExecutor, client client.Client, clientSet kubernetes.Interface, namespace string) *creator {
	return &creator{
		client:            client,
		sensitiveDoguRepo: repo,
		doguFetcher:       localDoguFetcher,
		executor:          commandExecutor,
		clientSet:         clientSet,
		apiClient:         &apiClient{},
		namespace:         namespace,
	}
}

// CreateAll creates all service accounts for a given dogu. Existing service accounts will be skipped.
func (c *creator) CreateAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	sensitiveConfig, err := c.sensitiveDoguRepo.Get(ctx, cescommons.SimpleName(dogu.GetSimpleName()))
	if err != nil {
		return fmt.Errorf("unbale to get sensitive config for dogu %s: %w", dogu.GetSimpleName(), err)
	}

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type

		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			lErr := c.createDoguServiceAccount(ctx, dogu, &sensitiveConfig, serviceAccount, registryCredentialPath)
			if lErr != nil {
				return fmt.Errorf("unable to create service account for dogu %s: %w", serviceAccount.Type, lErr)
			}
		case componentKind:
			lErr := c.createComponentServiceAccount(ctx, dogu, &sensitiveConfig, serviceAccount, registryCredentialPath)
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

func (c *creator) createDoguServiceAccount(ctx context.Context, dogu *core.Dogu, senDoguCfg *config.DoguConfig,
	serviceAccount core.ServiceAccount, registryCredentialPath string) error {
	logger := log.FromContext(ctx)

	if skip := serviceAccountExists(registryCredentialPath, *senDoguCfg); skip {
		return nil
	}

	enabled, err := c.doguFetcher.Enabled(ctx, serviceAccount.Type)
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

func (c *creator) create(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, senDoguCfg *config.DoguConfig) error {
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

func serviceAccountExists(registryCredentialPath string, senDoguCfg config.DoguConfig) bool {
	entries := senDoguCfg.GetAll()

	for key := range entries {
		if strings.HasPrefix(key.String(), registryCredentialPath+"/") {
			return true
		}
	}

	return false
}

func getPodForServiceAccountDogu(ctx context.Context, client client.Client, saDogu *core.Dogu) (*corev1.Pod, error) {
	versionlessDoguLabel := map[string]string{v2.DoguLabelName: saDogu.GetSimpleName()}
	return v2.GetPodForLabels(ctx, client, versionlessDoguLabel)
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
	buffer, err := c.executor.ExecCommandForPod(ctx, saPod, command, exec.PodReady)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	saCreds, err := c.parseServiceCommandOutput(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account: %w", err)
	}

	return saCreds, nil
}

func (c *creator) writeServiceAccounts(ctx context.Context, senDoguCfg *config.DoguConfig, serviceAccount core.ServiceAccount, saCreds map[string]string) error {
	for key, value := range saCreds {
		path := "/sa-" + serviceAccount.Type + "/" + key

		updatedCfg, err := senDoguCfg.Set(config.Key(path), config.Value(value))
		if err != nil {
			return fmt.Errorf("failed to set sa value of key %s to sensisitive dogu config: %w", key, err)
		}

		senDoguCfg.Config = updatedCfg
	}

	if err := writeConfig(ctx, senDoguCfg, c.sensitiveDoguRepo); err != nil {
		return fmt.Errorf("failed write config for dogu %s after updating: %w", senDoguCfg.DoguName, err)
	}

	return nil
}

func writeConfig(ctx context.Context, senDoguCfg *config.DoguConfig, cfgRepo sensitiveDoguConfigRepository) error {
	update, err := cfgRepo.Update(ctx, *senDoguCfg)
	if err != nil {
		if regLibErr.IsConflictError(err) {
			mergedCfg, lErr := cfgRepo.SaveOrMerge(ctx, *senDoguCfg)
			if lErr != nil {
				return fmt.Errorf("unable to save and merge sensitive config for dogu %s after conflict error: %w", senDoguCfg.DoguName, lErr)
			}

			*senDoguCfg = mergedCfg

			return nil
		}

		return fmt.Errorf("unable to update sensitive config for dogu %s: %w", senDoguCfg.DoguName, err)
	}

	*senDoguCfg = update

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
