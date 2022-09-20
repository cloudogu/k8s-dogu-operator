package serviceaccount

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"io"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/cesapp/v5/config"
	"github.com/cloudogu/cesapp/v5/keys"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// commandExecutor is used to execute command in a dogu
type commandExecutor interface {
	ExecCommand(ctx context.Context, targetDogu string, namespace string, command *core.ExposedCommand, params []string) (*bytes.Buffer, error)
}

type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(doguName string) (installedDogu *core.Dogu, err error)
}

// creator is the unit to handle the creation of service accounts
type creator struct {
	registry    registry.Registry
	doguFetcher localDoguFetcher
	executor    commandExecutor
}

// NewCreator creates a new instance of ServiceAccountCreator
func NewCreator(registry registry.Registry, commandExecutor commandExecutor) *creator {
	localFetcher := cesregistry.NewLocalDoguFetcher(registry.DoguRegistry())
	return &creator{
		registry:    registry,
		doguFetcher: localFetcher,
		executor:    commandExecutor,
	}
}

// CreateAll creates all service accounts for a given dogu. Existing service accounts will be skipped.
func (c *creator) CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type
		doguConfig := c.registry.DoguConfig(dogu.GetSimpleName())

		exists, err := serviceAccountExists(registryCredentialPath, doguConfig)
		if err != nil {
			return err
		}

		if exists {
			logger.Info("skipping creation of service account because it already exists")
			continue
		}

		doguRegistry := c.registry.DoguRegistry()
		enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
		}

		if !enabled && c.isOptionalServiceAccount(dogu, serviceAccount.Type) {
			logger.Info(fmt.Sprintf("skipping optional service account creation for %s, because the dogu is not installed", serviceAccount.Type))
			continue
		}

		if !enabled && c.containsDependency(dogu.Dependencies, serviceAccount.Type) {
			return fmt.Errorf("service account dogu is not enabled and not optional")
		}

		saDogu, err := c.doguFetcher.FetchInstalled(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to get service account dogu.json: %w", err)
		}

		saCreds, err := c.executeCommand(ctx, dogu, saDogu, namespace, serviceAccount)
		if err != nil {
			return fmt.Errorf("failed to execute service account create command: %w", err)
		}

		err = c.saveServiceAccount(serviceAccount, doguConfig, saCreds)
		if err != nil {
			return fmt.Errorf("failed to save the service account credentials: %w", err)
		}
	}

	return nil
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

func (c *creator) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, namespace string, serviceAccount core.ServiceAccount) (map[string]string, error) {
	createCommand, err := getCommand(saDogu, "service-account-create")
	if err != nil {
		return nil, err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())

	buffer, err := c.executor.ExecCommand(ctx, saDogu.GetSimpleName(), namespace, createCommand, args)
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
	keyProvider, err := keys.NewKeyProvider(config.Keys{Type: keyProviderStr})
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

func getCommand(dogu *core.Dogu, command string) (*core.ExposedCommand, error) {
	for _, cmd := range dogu.ExposedCommands {
		if cmd.Name == command {
			return &core.ExposedCommand{
				Name:        cmd.Name,
				Description: cmd.Description,
				Command:     cmd.Command,
			}, nil
		}
	}

	return nil, fmt.Errorf("service account dogu %s does not expose %s command", dogu.GetSimpleName(), command)
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

func serviceAccountExists(saPath string, doguConfig registry.ConfigurationContext) (bool, error) {
	exists, err := doguConfig.Exists(saPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if service account already exists")
	}
	if exists {
		return true, nil
	}
	return false, nil
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
