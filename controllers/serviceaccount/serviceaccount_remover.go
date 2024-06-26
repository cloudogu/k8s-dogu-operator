package serviceaccount

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/dogu"

	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// Remover removes a dogu's service account.
type remover struct {
	client                   client.Client
	sensitiveDoguCfgProvider SensitiveDoguConfigProvider
	doguFetcher              cloudogu.LocalDoguFetcher
	localDoguRegistry        dogu.LocalRegistry
	executor                 cloudogu.CommandExecutor
	clientSet                kubernetes.Interface
	apiClient                serviceAccountApiClient
	namespace                string
}

// NewRemover creates a new instance of ServiceAccountRemover
func NewRemover(provider SensitiveDoguConfigProvider, localFetcher cloudogu.LocalDoguFetcher, localDoguRegistry dogu.LocalRegistry, commandExecutor cloudogu.CommandExecutor, client client.Client, clientSet kubernetes.Interface, namespace string) *remover {
	return &remover{
		client:                   client,
		sensitiveDoguCfgProvider: provider,
		doguFetcher:              localFetcher,
		localDoguRegistry:        localDoguRegistry,
		executor:                 commandExecutor,
		clientSet:                clientSet,
		apiClient:                &apiClient{},
		namespace:                namespace,
	}
}

// RemoveAll removes all service accounts for a given dogu
func (r *remover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	senDoguCfg, err := r.sensitiveDoguCfgProvider.GetSensitiveDoguConfig(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("unable to get sensitive dogu config for dogu %s: %w", dogu.GetSimpleName(), err)
	}

	var allProblems error
	for _, serviceAccount := range dogu.ServiceAccounts {
		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			lErr := r.removeDoguServiceAccount(ctx, dogu, serviceAccount, senDoguCfg)
			if lErr != nil {
				allProblems = errors.Join(allProblems, fmt.Errorf("unable to remove service account for dogu %s: %w", serviceAccount.Type, lErr))
			}
		case componentKind:
			lErr := r.removeComponentServiceAccount(ctx, dogu, serviceAccount, senDoguCfg)
			if lErr != nil {
				allProblems = errors.Join(allProblems, fmt.Errorf("unable to remove service account for component %s: %w", serviceAccount.Type, lErr))
			}
		default:
			logger.Error(fmt.Errorf("unknown service account kind: %s", serviceAccount.Kind), "skipping service account deletion")
			continue
		}
	}

	return allProblems
}

func (r *remover) removeDoguServiceAccount(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, senDoguCfg SensitiveDoguConfig) error {
	logger := log.FromContext(ctx)
	registryCredentialPath := "sa-" + serviceAccount.Type

	exists, err := serviceAccountExists(ctx, registryCredentialPath, senDoguCfg)
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

	err = r.delete(ctx, serviceAccount, dogu, senDoguCfg, registryCredentialPath)
	if err != nil {
		return err
	}

	return nil
}

func (r *remover) delete(
	ctx context.Context,
	serviceAccount core.ServiceAccount,
	dogu *core.Dogu,
	deleter SensitiveDoguConfigDeleter,
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

	err = deleter.DeleteRecursive(ctx, registryCredentialPath)
	if err != nil {
		return fmt.Errorf("failed to remove service account from sensitive config: %w", err)
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
