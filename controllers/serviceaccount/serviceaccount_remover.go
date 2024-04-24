package serviceaccount

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/localregistry"
	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"

	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// Remover removes a dogu's service account.
type remover struct {
	client            client.Client
	registry          registry.Registry
	doguFetcher       cloudogu.LocalDoguFetcher
	localDoguRegistry localregistry.LocalDoguRegistry
	executor          cloudogu.CommandExecutor
	clientSet         kubernetes.Interface
	apiClient         serviceAccountApiClient
	namespace         string
}

// NewRemover creates a new instance of ServiceAccountRemover
func NewRemover(registry registry.Registry, localFetcher cloudogu.LocalDoguFetcher, localDoguRegistry localregistry.LocalDoguRegistry, commandExecutor cloudogu.CommandExecutor, client client.Client, clientSet kubernetes.Interface, namespace string) *remover {
	return &remover{
		client:            client,
		registry:          registry,
		doguFetcher:       localFetcher,
		localDoguRegistry: localDoguRegistry,
		executor:          commandExecutor,
		clientSet:         clientSet,
		apiClient:         &apiClient{},
		namespace:         namespace,
	}
}

// RemoveAll removes all service accounts for a given dogu
func (r *remover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	var allProblems error
	for _, serviceAccount := range dogu.ServiceAccounts {
		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			err := r.removeDoguServiceAccount(ctx, dogu, serviceAccount)
			if err != nil {
				allProblems = errors.Join(allProblems, err)
			}
		case componentKind:
			err := r.removeComponentServiceAccount(ctx, dogu, serviceAccount)
			if err != nil {
				allProblems = errors.Join(allProblems, err)
			}
		case cesKind:
			if serviceAccount.Type == "cesappd" {
				err := r.removeCesControlServiceAccount(dogu)
				if err != nil {
					allProblems = errors.Join(allProblems, err)
				}
			}
			continue
		default:
			logger.Error(fmt.Errorf("unknown service account kind: %s", serviceAccount.Kind), "skipping service account deletion")
			continue
		}
	}

	return allProblems
}

func (r *remover) removeDoguServiceAccount(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount) error {
	logger := log.FromContext(ctx)
	registryCredentialPath := "sa-" + serviceAccount.Type
	doguConfig := r.registry.DoguConfig(dogu.GetSimpleName())

	exists, err := serviceAccountExists(registryCredentialPath, doguConfig)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("skipping removal of service account because it does not exists")
		return nil
	}

	enabled, err := r.localDoguRegistry.IsEnabled(ctx, serviceAccount.Type)
	if err != nil {
		return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
	}
	if !enabled {
		logger.Info("skipping removal of service account because dogu is not enabled")
		return nil
	}

	err = r.delete(ctx, serviceAccount, dogu, doguConfig, registryCredentialPath)
	if err != nil {
		return err
	}

	return nil
}

func (r *remover) delete(
	ctx context.Context,
	serviceAccount core.ServiceAccount,
	dogu *core.Dogu,
	doguConfig registry.ConfigurationContext,
	registryCredentialPath string,
) error {
	saDogu, err := r.doguFetcher.FetchInstalled(ctx, serviceAccount.Type)
	if err != nil {
		return fmt.Errorf("failed to get service account dogu.json: %w", err)
	}

	serviceAccountPod, err := getPodForServiceAccountDogu(ctx, r.client, saDogu)
	if err != nil {
		return fmt.Errorf("could not find service account producer pod %s: %w", saDogu.GetSimpleName(), err)
	}

	err = r.executeCommand(ctx, dogu, saDogu, serviceAccountPod, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to execute service account remove command: %w", err)
	}

	err = doguConfig.DeleteRecursive(registryCredentialPath)
	if err != nil {
		return fmt.Errorf("failed to remove service account from config: %w", err)
	}

	return nil
}

func (r *remover) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, saPod *v1.Pod, serviceAccount core.ServiceAccount) error {
	removeCommand, err := getExposedCommand(saDogu, "service-account-remove")
	if err != nil {
		return err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())

	command := exec.NewShellCommand(removeCommand.Command, args...)
	_, err = r.executor.ExecCommandForPod(ctx, saPod, command, cloudogu.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

func (r *remover) removeCesControlServiceAccount(dogu *core.Dogu) error {
	hostConfig := r.registry.HostConfig(k8sCesControl)
	exists, err := hostConfig.Exists(dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to read host config for dogu %s", dogu.GetSimpleName())
	}

	if exists {
		err = hostConfig.DeleteRecursive(dogu.GetSimpleName())
		if err != nil {
			return fmt.Errorf("failed to delete host config for dogu %s", dogu.GetSimpleName())
		}
	}

	return nil
}
