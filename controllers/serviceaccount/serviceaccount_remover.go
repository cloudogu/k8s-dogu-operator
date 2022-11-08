package serviceaccount

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"

	"github.com/hashicorp/go-multierror"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
)

// Remover removes a dogu's service account.
type remover struct {
	client      client.Client
	registry    registry.Registry
	doguFetcher localDoguFetcher
	executor    commandExecutor
}

// NewRemover creates a new instance of ServiceAccountRemover
func NewRemover(registry registry.Registry, commandExecutor commandExecutor, client client.Client) *remover {
	localFetcher := cesregistry.NewLocalDoguFetcher(registry.DoguRegistry())
	return &remover{
		client:      client,
		registry:    registry,
		doguFetcher: localFetcher,
		executor:    commandExecutor,
	}
}

// RemoveAll removes all service accounts for a given dogu
func (r *remover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	var allProblems error

	for _, serviceAccount := range dogu.ServiceAccounts {
		registryCredentialPath := "sa-" + serviceAccount.Type
		doguConfig := r.registry.DoguConfig(dogu.GetSimpleName())

		exists, err := serviceAccountExists(registryCredentialPath, doguConfig)
		if err != nil {
			allProblems = multierror.Append(allProblems, err)
			continue
		}

		if !exists {
			logger.Info("skipping removal of service account because it does not exists")
			continue
		}

		doguRegistry := r.registry.DoguRegistry()
		enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
		if err != nil {
			allProblems = multierror.Append(allProblems, fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err))
			continue
		}
		if !enabled {
			logger.Info("skipping removal of service account because dogu is not enabled")
			continue
		}

		saDogu, err := r.doguFetcher.FetchInstalled(serviceAccount.Type)
		if err != nil {
			allProblems = multierror.Append(allProblems, fmt.Errorf("failed to get service account dogu.json: %w", err))
			continue
		}

		serviceAccountPod, err := getPodForServiceAccountDogu(ctx, r.client, saDogu)
		if err != nil {
			return fmt.Errorf("could not find service account producer pod %s: %w", saDogu.GetSimpleName(), err)
		}

		err = r.executeCommand(ctx, dogu, saDogu, serviceAccountPod, serviceAccount)
		if err != nil {
			allProblems = multierror.Append(allProblems, fmt.Errorf("failed to execute service account remove command: %w", err))
			continue
		}

		err = doguConfig.DeleteRecursive(registryCredentialPath)
		if err != nil {
			allProblems = multierror.Append(allProblems, fmt.Errorf("failed to remove service account from config: %w", err))
			continue
		}
	}

	return allProblems
}

func (r *remover) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, saPod *v1.Pod, serviceAccount core.ServiceAccount) error {
	removeCommand, err := getExposedCommand(saDogu, "service-account-remove")
	if err != nil {
		return err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())

	command := &exec.ShellCommand{Command: removeCommand.Command, Args: args}
	_, err = r.executor.ExecCommandForPod(ctx, saPod, command, exec.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
