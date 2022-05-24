package serviceaccount

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/registry"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Remover removes a dogu's service account.
type remover struct {
	registry registry.Registry
	executor commandExecutor
}

// NewRemover creates a new instance of ServiceAccountRemover
func NewRemover(registry registry.Registry, commandExecutor commandExecutor) *remover {
	return &remover{
		registry: registry,
		executor: commandExecutor,
	}
}

// RemoveAll removes all service accounts for a given dogu
func (r *remover) RemoveAll(ctx context.Context, namespace string, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type
		doguConfig := r.registry.DoguConfig(dogu.GetSimpleName())

		exists, err := serviceAccountExists(registryCredentialPath, doguConfig)
		if err != nil {
			return err
		}

		if !exists {
			logger.Info("skipping removal of service account because it does not exists")
			continue
		}

		doguRegistry := r.registry.DoguRegistry()
		enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
		}
		if !enabled {
			logger.Info("skipping removal of service account because dogu is not enabled")
			continue
		}

		saDogu, err := doguRegistry.Get(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to get service account dogu.json: %w", err)
		}

		err = r.executeCommand(ctx, dogu, saDogu, namespace, serviceAccount)
		if err != nil {
			return fmt.Errorf("failed to execute service account remove command: %w", err)
		}

		err = doguConfig.DeleteRecursive(registryCredentialPath)
		if err != nil {
			return fmt.Errorf("failed to remove service account from config: %w", err)
		}
	}

	return nil
}

func (r *remover) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, namespace string, serviceAccount core.ServiceAccount) error {
	removeCommand, err := getCommand(saDogu, "service-account-remove")
	if err != nil {
		return err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())
	_, err = r.executor.ExecCommand(ctx, saDogu.GetSimpleName(), namespace, removeCommand, args)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
