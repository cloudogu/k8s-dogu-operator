package serviceaccount

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// remover is the unit to handle the removal of service accounts
type remover struct {
	registry registry.Registry
	executor commandExecutor
}

// NewServiceAccountRemover creates a new instance of ServiceAccountCreator
func NewServiceAccountRemover(registry registry.Registry, commandExecutor commandExecutor) *remover {
	return &remover{
		registry: registry,
		executor: commandExecutor,
	}
}

// RemoveServiceAccounts removes all service accounts for a given dogu
func (r *remover) RemoveServiceAccounts(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		parentPath := "sa-" + serviceAccount.Type
		doguConfig := r.registry.DoguConfig(dogu.GetSimpleName())

		// check if the service account really exists
		exists, err := serviceAccountExists(parentPath, doguConfig)
		if err != nil {
			return err
		}

		if !exists {
			logger.Info("skipping removal of service account because it does not exists")
			continue
		}

		// check if the dogu is enabled
		doguRegistry := r.registry.DoguRegistry()
		enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
		}
		if !enabled {
			logger.Info("skipping removal of service account because dogu is not enabled")
			continue
		}

		// get the service account dogu.json
		targetDescriptor, err := doguRegistry.Get(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to get service account dogu.json: %w", err)
		}

		// execute remove service account command
		err = r.executeServiceAccountRemoveCommand(ctx, dogu, targetDescriptor, doguResource.Namespace, serviceAccount)
		if err != nil {
			return fmt.Errorf("failed to execute service account remove command: %w", err)
		}

		// remove credentials
		err = doguConfig.DeleteRecursive(parentPath)
		if err != nil {
			return fmt.Errorf("failed to remove service account from config: %w", err)
		}
	}

	return nil
}

func (r *remover) executeServiceAccountRemoveCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, namespace string, serviceAccount core.ServiceAccount) error {
	removeCommand := r.getRemoveCommand(saDogu)
	if removeCommand == nil {
		return fmt.Errorf("service account dogu does not expose remove command")
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())
	_, err := r.executor.ExecCommand(ctx, saDogu.GetSimpleName(), namespace, removeCommand, args)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

func (r *remover) getRemoveCommand(dogu *core.Dogu) *core.ExposedCommand {
	var createCmd *core.ExposedCommand
	for _, cmd := range dogu.ExposedCommands {
		if cmd.Name == "service-account-remove" {
			createCmd = &core.ExposedCommand{
				Name:        cmd.Name,
				Description: cmd.Description,
				Command:     cmd.Command,
			}
			break
		}
	}

	return createCmd
}
