package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExperimentalHelmfileStep struct {
	isEnabled            bool
	doguFetcher          resourceDoguFetcher
	helmfileGlobalConfig config.HelmfileGlobalConfig
	openDeskDoguConfig   config.HelmfileOpenDeskConfig
	client               k8sClient
}

func NewExperimentalHelmfileStep(
	config *config.OperatorConfig,
	fetcher cesregistry.ResourceDoguFetcher,
	client client.Client,
) *ExperimentalHelmfileStep {
	return &ExperimentalHelmfileStep{
		isEnabled:            config.ExperimentalHelmfileSupport,
		doguFetcher:          fetcher,
		helmfileGlobalConfig: config.HelmfileGlobalConfig,
		openDeskDoguConfig:   config.HelmfileOpenDeskConfig,
		client:               client,
	}
}

func (e *ExperimentalHelmfileStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).
		WithName("experimentalHelmfileStep").
		WithValues("dogu", resource.Name)

	if !e.isEnabled {
		return steps.Continue()
	}

	doguDescriptor, _, err := e.doguFetcher.FetchWithResource(ctx, resource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguDescriptor.Properties["kind"] != "Helmfile" {
		logger.Info("dogu is not a helmfile dogu, continuing")
		return steps.Continue()
	}

	logger.Info("handling helmfile dogu")

	if resource.Name != "opendesk" {
		return steps.RequeueWithError(fmt.Errorf("openDesk is currently the only helmfile dogu supported"))
	}

	applyOptions, err := e.extractAndConfigureHelmfile(ctx)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to extract and configure helmfile: %w", err))
	}

	err = applyOptions.command(ctx).Run()
	if err != nil {
		var stdErr string
		var exitError *exec.ExitError
		isExitErr := errors.As(err, &exitError)
		if isExitErr {
			stdErr = string(exitError.Stderr)
		}
		return steps.RequeueWithError(fmt.Errorf("failed to apply helmfile of %s: %w: stderr: %s", resource.Name, err, stdErr))
	}

	return steps.Abort()
}

func (e *ExperimentalHelmfileStep) extractAndConfigureHelmfile(ctx context.Context) (helmfileApplyOptions, error) {
	// TODO
	// download helmfile from source url to unpackDir
	// extract tar.gz to <unpackDir>/opendesk
	// read domain from configmap
	// write domain as YAML under key global.domain in <unpackDir>/opendesk/<valuesFilePath>
	// generate masterpassword to secret if not exists, otherwise read from secret
	// set MASTER_PASSWORD env
	// write apply options
	panic("not implemented")
}

type helmfileApplyOptions struct {
	config.HelmfileGlobalConfig
	helmfilePath string
	environment  string
	envVars      []string
	namespace    string
}

func (h helmfileApplyOptions) command(ctx context.Context) *exec.Cmd {
	helmfilePath := filepath.Join(h.HelmfileUnpackDir, h.helmfilePath)
	cmd := exec.CommandContext(ctx,
		h.HelmfileBin, "apply",
		"--environment", h.environment,
		"--namespace", h.namespace,
		"--helm-binary", h.HelmBin,
		"--quiet", "--no-color")
	envWithHelmDirs := append(h.envVars,
		fmt.Sprintf("HELM_PLUGINS=%s", h.HelmPluginHome),
		fmt.Sprintf("HELM_CACHE_HOME=%s", h.HelmCacheHome),
		fmt.Sprintf("HELM_CONFIG_HOME=%s", h.HelmConfigHome),
		fmt.Sprintf("HELM_DATA_HOME=%s", h.HelmDataHome))
	cmd.Env = append(os.Environ(), envWithHelmDirs...)
	cmd.Dir = helmfilePath
	return cmd
}
