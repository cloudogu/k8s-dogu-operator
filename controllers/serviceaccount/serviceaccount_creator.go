package serviceaccount

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/keys"
	"github.com/cloudogu/cesapp-lib/registry"

	"github.com/pkg/errors"
)

// doguKind describes a service account on a dogu.
const (
	doguKind = "dogu"
	cesKind  = "ces"
)

const (
	cesappd       = "cesappd"
	k8sCesControl = "k8s-ces-control"
)

// creator is the unit to handle the creation of service accounts
type creator struct {
	client      client.Client
	registry    registry.Registry
	doguFetcher cloudogu.LocalDoguFetcher
	executor    cloudogu.CommandExecutor
}

// NewCreator creates a new instance of ServiceAccountCreator
func NewCreator(registry registry.Registry, commandExecutor cloudogu.CommandExecutor, client client.Client) *creator {
	localFetcher := cesregistry.NewLocalDoguFetcher(registry.DoguRegistry())
	return &creator{
		client:      client,
		registry:    registry,
		doguFetcher: localFetcher,
		executor:    commandExecutor,
	}
}

// CreateAll creates all service accounts for a given dogu. Existing service accounts will be skipped.
func (c *creator) CreateAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type
		doguConfig := c.registry.DoguConfig(dogu.GetSimpleName())

		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			err := c.createDoguServiceAccount(ctx, dogu, doguConfig, serviceAccount, registryCredentialPath)
			if err != nil {
				return err
			}
		case cesKind:
			if serviceAccount.Type == cesappd {
				err := c.createCesControlServiceAccount(ctx, dogu, doguConfig, serviceAccount, registryCredentialPath)
				if err != nil {
					return err
				}
			}
			continue
		default:
			logger.Error(fmt.Errorf("unknown service account kind: %s", serviceAccount.Kind), "skipping service account creation")
			continue
		}
	}

	return nil
}

func (c *creator) createDoguServiceAccount(ctx context.Context, dogu *core.Dogu, doguConfig registry.ConfigurationContext,
	serviceAccount core.ServiceAccount, registryCredentialPath string) error {
	logger := log.FromContext(ctx)
	skip, err := serviceAccountExists(registryCredentialPath, doguConfig)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	doguRegistry := c.registry.DoguRegistry()
	enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
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

	err = c.create(ctx, dogu, serviceAccount, doguConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *creator) create(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, doguConfig registry.ConfigurationContext) error {
	saDogu, err := c.doguFetcher.FetchInstalled(serviceAccount.Type)
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

	err = c.saveServiceAccount(serviceAccount, doguConfig, saCreds)
	if err != nil {
		return fmt.Errorf("failed to save the service account credentials: %w", err)
	}
	return nil
}

func (c *creator) createCesControlServiceAccount(ctx context.Context, dogu *core.Dogu,
	doguConfig registry.ConfigurationContext, sa core.ServiceAccount, registryCredentialPath string) error {
	logger := log.FromContext(ctx)

	skip, err := serviceAccountExists(registryCredentialPath, doguConfig)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	exists, err := c.registry.HostConfig(k8sCesControl).Exists(dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if service account already exists: %w", err)
	}
	if exists {
		logger.Info("%s service account for dogu %s already exists", sa.Type, dogu.GetSimpleName())
		return nil
	}

	labels := map[string]string{"app": k8sCesControl}
	pod, err := v1.GetPodForLabels(ctx, c.client, labels)
	if err != nil && c.isOptionalServiceAccount(dogu, sa.Type) {
		logger.Info("Skipping creation of service account % because the pod was not found and the service "+
			"account is optional", sa.Type)
		return nil
	}
	if err != nil && !c.isOptionalServiceAccount(dogu, sa.Type) {
		return fmt.Errorf("failed to get pod for labels %v: %w", labels, err)
	}

	var cmdParams []string
	cmdParams = append(cmdParams, "service-account-create")
	cmdParams = append(cmdParams, dogu.GetSimpleName())
	cmdParams = append(cmdParams, sa.Params...)
	command := exec.NewShellCommand(fmt.Sprintf("/%s/%s", k8sCesControl, k8sCesControl), cmdParams...)

	buffer, err := c.executor.ExecCommandForPod(ctx, pod, command, cloudogu.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to exec command [%s] for pod %s: %w", command.String(), pod.Name, err)
	}

	saCreds, err := c.parseServiceCommandOutput(buffer)
	if err != nil {
		return fmt.Errorf("failed to parse service account: %w", err)
	}

	err = c.saveServiceAccount(sa, doguConfig, saCreds)
	if err != nil {
		return fmt.Errorf("failed to save service account: %w", err)
	}

	return nil
}

func serviceAccountExists(registryCredentialPath string,
	doguConfig registry.ConfigurationContext) (bool, error) {

	exists, err := doguConfig.Exists(registryCredentialPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if service account already exists")
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

func (c *creator) saveServiceAccount(serviceAccount core.ServiceAccount, doguConfig registry.ConfigurationContext, credentials map[string]string) error {
	publicKey, err := c.getPublicKey(doguConfig)
	if err != nil {
		return fmt.Errorf("failed to read public key from string: %w", err)
	}

	err = c.writeServiceAccounts(doguConfig, serviceAccount, credentials, publicKey)
	if err != nil {
		return fmt.Errorf("failed to write service account: %w", err)
	}

	return nil
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

func (c *creator) getPublicKey(doguConfig registry.ConfigurationContext) (*keys.PublicKey, error) {
	keyProviderStr, err := c.registry.GlobalConfig().Get("key_provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get key provider: %w", err)
	}
	keyProvider, err := keys.NewKeyProvider(keyProviderStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyprovider: %w", err)
	}
	pubKeyStr, err := doguConfig.Get(registry.KeyDoguPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu public key: %w", err)
	}
	publicKey, err := keyProvider.ReadPublicKeyFromString(pubKeyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key from string: %w", err)
	}

	return publicKey, nil
}

func (c *creator) writeServiceAccounts(doguConfig registry.ConfigurationContext, serviceAccount core.ServiceAccount, saCreds map[string]string, publicKey *keys.PublicKey) error {
	for key, value := range saCreds {
		path := "/sa-" + serviceAccount.Type + "/" + key

		encrypted, err := publicKey.Encrypt(value)
		if err != nil {
			return fmt.Errorf("failed to encrypt service account value of key %s: %w", key, err)
		}

		err = doguConfig.Set(path, encrypted)
		if err != nil {
			return fmt.Errorf("failed to set encrypted sa value of key %s: %w", key, err)
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
