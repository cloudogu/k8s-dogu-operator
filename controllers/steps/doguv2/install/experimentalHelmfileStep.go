package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
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

	applyOptions, err := e.extractAndConfigureHelmfile(ctx, resource.Name)
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

func (e *ExperimentalHelmfileStep) extractAndConfigureHelmfile(ctx context.Context, name string) (helmfileApplyOptions, error) {
	var options helmfileApplyOptions
	cfg := e.openDeskDoguConfig
	if e.helmfileGlobalConfig.HelmfileUnpackDir == "" || cfg.Namespace == "" ||
		cfg.DomainConfigMap.Name == "" || cfg.DomainConfigMap.Key == "" ||
		cfg.MasterkeySecret.Name == "" || cfg.MasterkeySecret.Key == "" || !filepath.IsLocal(cfg.ValuesFilePath) {
		return options, fmt.Errorf("helmfile unpack directory, namespace, ConfigMap and Secret references, and a local values path are required")
	}

	if err := e.downloadHelmfile(ctx, name); err != nil {
		return options, fmt.Errorf("download and extract helmfile: %w", err)
	}

	baseDir := filepath.Join(e.helmfileGlobalConfig.HelmfileUnpackDir, name)
	workingDir, err := helmfileWorkingDir(baseDir)
	if err != nil {
		return options, err
	}

	var domainConfig corev1.ConfigMap
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: cfg.Namespace, Name: cfg.DomainConfigMap.Name}, &domainConfig); err != nil {
		return options, fmt.Errorf("read domain ConfigMap: %w", err)
	}

	domain := domainConfig.Data[cfg.DomainConfigMap.Key]
	if strings.TrimSpace(domain) == "" {
		return options, fmt.Errorf("domain ConfigMap %s has missing or empty key %s", cfg.DomainConfigMap.Name, cfg.DomainConfigMap.Key)
	}

	// A YAML round trip copies nested maps without changing the supplied configuration.
	valuesYAML, err := yaml.Marshal(cfg.ExtraValues)
	if err != nil {
		return options, fmt.Errorf("marshal extra values: %w", err)
	}

	values := make(map[string]any)
	if err := yaml.Unmarshal(valuesYAML, &values); err != nil {
		return options, fmt.Errorf("copy extra values: %w", err)
	}

	if values == nil {
		values = make(map[string]any)
	}
	global, ok := values["global"].(map[string]any)
	if !ok {
		if values["global"] != nil {
			return options, fmt.Errorf("extra values global must be a mapping")
		}

		global = make(map[string]any)
	}
	global["domain"] = domain
	values["global"] = global
	valuesYAML, err = yaml.Marshal(values)
	if err != nil {
		return options, fmt.Errorf("marshal helmfile values: %w", err)
	}

	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return options, fmt.Errorf("open helmfile directory: %w", err)
	}

	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(cfg.ValuesFilePath), 0755); err != nil {
		return options, fmt.Errorf("create values directory: %w", err)
	}

	if err := root.WriteFile(cfg.ValuesFilePath, valuesYAML, 0644); err != nil {
		return options, fmt.Errorf("write helmfile values: %w", err)
	}

	password, err := e.masterPassword(ctx)
	if err != nil {
		return options, err
	}

	keys := make([]string, 0, len(cfg.ExtraEnvVars))
	for key := range cfg.ExtraEnvVars {
		if key != "MASTER_PASSWORD" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		env = append(env, key+"="+cfg.ExtraEnvVars[key])
	}
	env = append(env, "MASTER_PASSWORD="+password)

	return helmfileApplyOptions{
		HelmfileGlobalConfig: e.helmfileGlobalConfig,
		helmfilePath:         filepath.Join(name, workingDir),
		environment:          cfg.Environment,
		namespace:            cfg.Namespace,
		envVars:              env,
	}, nil
}

func (e *ExperimentalHelmfileStep) downloadHelmfile(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.openDeskDoguConfig.HelmfileSource, nil)
	if err != nil {
		return err
	}

	response, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status %s", response.Status)
	}

	unpackDir := e.helmfileGlobalConfig.HelmfileUnpackDir
	if err := os.MkdirAll(unpackDir, 0755); err != nil {
		return err
	}

	stagingDir, err := os.MkdirTemp(unpackDir, fmt.Sprintf(".%s-", name))
	if err != nil {
		return err
	}

	defer os.RemoveAll(stagingDir)
	if err := extractHelmfileArchive(ctx, response.Body, stagingDir); err != nil {
		return err
	}

	if _, err := helmfileWorkingDir(stagingDir); err != nil {
		return err
	}

	destination := filepath.Join(unpackDir, name)
	if err := os.RemoveAll(destination); err != nil {
		return err
	}

	return os.Rename(stagingDir, destination)
}

func extractHelmfileArchive(ctx context.Context, source io.Reader, destination string) error {
	compressed, err := gzip.NewReader(source)
	if err != nil {
		return err
	}

	defer compressed.Close()
	archive := tar.NewReader(compressed)
	var archiveRoot string
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			// Consume the gzip trailer so checksum and truncation errors are reported.
			_, err = io.Copy(io.Discard, compressed)
			return err
		}

		if err != nil {
			return err
		}

		name := strings.TrimPrefix(header.Name, "./")
		parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
		if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}

		for _, part := range parts {
			if part == ".." || part == "." || part == "" {
				return fmt.Errorf("unsafe archive path %q", header.Name)
			}
		}
		if archiveRoot == "" {
			archiveRoot = parts[0]
		}
		if parts[0] != archiveRoot {
			return fmt.Errorf("archive contains multiple root directories")
		}

		target := filepath.Join(destination, filepath.Join(parts[1:]...))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if len(parts) < 2 {
				return fmt.Errorf("archive must contain a root directory")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(file, archive)
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry %q (type %d)", header.Name, header.Typeflag)
		}
	}
}

func helmfileWorkingDir(baseDir string) (string, error) {
	for _, directory := range []string{".", "helmfile"} {
		for _, name := range []string{"helmfile.yaml", "helmfile.yaml.gotmpl", "helmfile.yml", "helmfile.yml.gotmpl"} {
			info, err := os.Stat(filepath.Join(baseDir, directory, name))
			if err == nil && info.Mode().IsRegular() {
				return directory, nil
			}

			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}
	}
	return "", fmt.Errorf("no helmfile entrypoint found in extracted root or helmfile subdirectory")
}

func (e *ExperimentalHelmfileStep) masterPassword(ctx context.Context) (string, error) {
	cfg := e.openDeskDoguConfig
	key := client.ObjectKey{Namespace: cfg.Namespace, Name: cfg.MasterkeySecret.Name}
	secret := &corev1.Secret{}
	err := e.client.Get(ctx, key, secret)
	if apierrors.IsNotFound(err) {
		var random [32]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate master password: %w", err)
		}
		secret = &corev1.Secret{
			Namespace: key.Namespace, Name: key.Name,
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{cfg.MasterkeySecret.Key: []byte(hex.EncodeToString(random[:]))},
		}
		err = e.client.Create(ctx, secret)
		if apierrors.IsAlreadyExists(err) {
			err = e.client.Get(ctx, key, secret)
		}
	}
	if err != nil {
		return "", fmt.Errorf("read or create master password Secret: %w", err)
	}

	password := string(secret.Data[cfg.MasterkeySecret.Key])
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("master password Secret %s has missing or empty key %s", key.Name, cfg.MasterkeySecret.Key)
	}

	return password, nil
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
