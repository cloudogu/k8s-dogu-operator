package serviceaccount

import (
	"context"
	"errors"
	"fmt"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-registry-lib/config"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
)

// Remover removes a dogu's service account.
type remover struct {
	client            client.Client
	sensitiveDoguRepo SensitiveDoguConfigRepository
	doguFetcher       localDoguFetcher
	executor          commandExecutor
	clientSet         kubernetes.Interface
	apiClient         serviceAccountApiClient
	namespace         string
}

// NewRemover creates a new instance of ServiceAccountRemover
func NewRemover(
	repo SensitiveDoguConfigRepository,
	localFetcher cesregistry.LocalDoguFetcher,
	commandExecutor exec.CommandExecutor,
	client client.Client,
	clientSet kubernetes.Interface,
	operatorConfig *opConfig.OperatorConfig,
) *remover {
	return &remover{
		client:            client,
		sensitiveDoguRepo: repo,
		doguFetcher:       localFetcher,
		executor:          commandExecutor,
		clientSet:         clientSet,
		apiClient:         &apiClient{},
		namespace:         operatorConfig.Namespace,
	}
}

// RemoveAll removes all service accounts for a given dogu
func (r *remover) RemoveAll(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	sensitiveConfig, err := r.sensitiveDoguRepo.Get(ctx, cescommons.SimpleName(dogu.GetSimpleName()))
	if err != nil {
		if cloudoguerrors.IsNotFoundError(err) {
			logger.Info(fmt.Sprintf("skipping service account removal because %s sensitive dogu config is not found", dogu.GetSimpleName()))
			return nil
		}
		return fmt.Errorf("unable to get sensitive config for dogu %s: %w", dogu.GetSimpleName(), err)
	}

	var allProblems error
	for _, serviceAccount := range dogu.ServiceAccounts {
		switch serviceAccount.Kind {
		case "":
			fallthrough
		case doguKind:
			lErr := r.removeDoguServiceAccount(ctx, dogu, serviceAccount, &sensitiveConfig)
			if lErr != nil {
				allProblems = errors.Join(allProblems, fmt.Errorf("unable to remove service account for dogu %s: %w", serviceAccount.Type, lErr))
			}
		case componentKind:
			lErr := r.removeComponentServiceAccount(ctx, dogu, serviceAccount, &sensitiveConfig)
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

// RemoveAllFromComponents removes all service accounts for a given dogu's components'
func (r *remover) RemoveAllFromComponents(ctx context.Context, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	sensitiveConfig, err := r.sensitiveDoguRepo.Get(ctx, cescommons.SimpleName(dogu.GetSimpleName()))
	if err != nil {
		if cloudoguerrors.IsNotFoundError(err) {
			logger.Info(fmt.Sprintf("skipping service account removal because %s sensitive dogu config is not found", dogu.GetSimpleName()))
			return nil
		}
		return fmt.Errorf("unable to get sensitive config for dogu %s: %w", dogu.GetSimpleName(), err)
	}

	for _, serviceAccount := range dogu.ServiceAccounts {
		if serviceAccount.Kind == componentKind {
			lErr := r.removeComponentServiceAccount(ctx, dogu, serviceAccount, &sensitiveConfig)
			if lErr != nil {
				err = errors.Join(err, fmt.Errorf("unable to remove service account for component %s: %w", serviceAccount.Type, lErr))
			}
		}
	}

	return err
}

func (r *remover) removeDoguServiceAccount(ctx context.Context, dogu *core.Dogu, serviceAccount core.ServiceAccount, senDoguCfg *config.DoguConfig) error {
	logger := log.FromContext(ctx)
	registryCredentialPath := "sa-" + serviceAccount.Type

	if exists := serviceAccountExists(registryCredentialPath, *senDoguCfg); !exists {
		logger.Info(fmt.Sprintf("skipping removal of service account '%s' because it does not exist", registryCredentialPath))

		return nil
	}

	enabled, err := r.doguFetcher.Enabled(ctx, cescommons.SimpleName(serviceAccount.Type))
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
	senDoguCfg *config.DoguConfig,
	registryCredentialPath string,
) error {
	saDogu, err := r.doguFetcher.FetchInstalled(ctx, cescommons.SimpleName(serviceAccount.Type))
	if err != nil {
		return fmt.Errorf("failed to get service account dogu.json: %w", err)
	}

	err = r.executeCommand(ctx, dogu, saDogu, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to execute service account remove command: %w", err)
	}

	updatedCfg := senDoguCfg.DeleteRecursive(config.Key(registryCredentialPath))
	senDoguCfg.Config = updatedCfg

	if lErr := writeConfig(ctx, senDoguCfg, r.sensitiveDoguRepo); lErr != nil {
		return fmt.Errorf("failed write config for dogu %s after updating: %w", senDoguCfg.DoguName, lErr)
	}

	return nil
}

func (r *remover) executeCommand(ctx context.Context, consumerDogu *core.Dogu, saDogu *core.Dogu, serviceAccount core.ServiceAccount) error {
	removeCommand, err := getExposedCommand(saDogu, "service-account-remove")
	if err != nil {
		return err
	}

	var args []string
	args = append(args, serviceAccount.Params...)
	args = append(args, consumerDogu.GetSimpleName())

	command := exec.NewShellCommand(removeCommand.Command, args...)

	doguResource, err := getDoguResource(ctx, saDogu.GetSimpleName(), r.namespace, r.client)
	if err != nil {
		return err
	}

	_, err = r.executor.ExecCommandForDogu(ctx, doguResource, command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
