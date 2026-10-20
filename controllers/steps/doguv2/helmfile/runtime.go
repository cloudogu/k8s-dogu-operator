package helmfile

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
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/retry-lib/retry"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const Finalizer = "k8s.cloudogu.com/helmfile-cleanup"
const ManagedAnnotation = "k8s.cloudogu.com/helmfile-managed"

// Runtime provides shared configuration, preparation and command execution for Helmfile steps.
type Runtime struct {
	GlobalConfig      config.HelmfileGlobalConfig
	OpenDeskConfig    config.HelmfileOpenDeskConfig
	Client            client.Client
	APIReader         client.Reader
	OperatorNamespace string
}

func NewRuntime(cfg *config.OperatorConfig, k8sClient client.Client, apiReader client.Reader) *Runtime {
	return &Runtime{
		GlobalConfig:      cfg.HelmfileGlobalConfig,
		OpenDeskConfig:    cfg.HelmfileOpenDeskConfig,
		Client:            k8sClient,
		APIReader:         apiReader,
		OperatorNamespace: cfg.Namespace,
	}
}

// Prepare recreates local files after a pod restart. Destruction must not
// recreate cluster prerequisites such as the namespace or the master password.
func (e *Runtime) Prepare(ctx context.Context, name string, installing bool) (Options, error) { //NOSONAR
	var options Options
	cfg := e.OpenDeskConfig
	if e.GlobalConfig.HelmfileUnpackDir == "" || cfg.Namespace == "" ||
		cfg.DomainConfigMap.Name == "" || cfg.DomainConfigMap.Key == "" ||
		cfg.MasterkeySecret.Name == "" || cfg.MasterkeySecret.Key == "" || !filepath.IsLocal(cfg.ValuesFilePath) {
		return options, fmt.Errorf("helmfile unpack directory, namespace, ConfigMap and Secret references, and a local values path are required")
	}

	if err := e.downloadHelmfile(ctx, name); err != nil {
		return options, fmt.Errorf("download and extract helmfile: %w", err)
	}

	baseDir := filepath.Join(e.GlobalConfig.HelmfileUnpackDir, name)
	workingDir, err := helmfileWorkingDir(baseDir)
	if err != nil {
		return options, err
	}

	var domainConfig corev1.ConfigMap
	if err := e.Client.Get(ctx, client.ObjectKey{Namespace: e.OperatorNamespace, Name: cfg.DomainConfigMap.Name}, &domainConfig); err != nil {
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

	password, err := e.getMasterPassword(ctx, installing)
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

	if installing {
		if err := e.createNamespaceIfNotExists(ctx, cfg.Namespace); err != nil {
			return options, fmt.Errorf("create deploy namespace if not exists: %w", err)
		}
	}

	return Options{
		HelmfileGlobalConfig: e.GlobalConfig,
		helmfilePath:         filepath.Join(name, workingDir),
		environment:          cfg.Environment,
		deployNamespace:      cfg.Namespace,
		envVars:              env,
	}, nil
}

func (e *Runtime) downloadHelmfile(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.OpenDeskConfig.HelmfileSource, nil)
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

	unpackDir := e.GlobalConfig.HelmfileUnpackDir
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

//nolint:funlen
func extractHelmfileArchive(ctx context.Context, source io.Reader, destination string) error { //NOSONAR
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

		if header.Typeflag == tar.TypeXGlobalHeader {
			// Global PAX metadata is not an extracted filesystem entry.
			continue
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
		case tar.TypeReg:
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

func (e *Runtime) getMasterPassword(ctx context.Context, create bool) (string, error) {
	cfg := e.OpenDeskConfig
	key := client.ObjectKey{Namespace: e.OperatorNamespace, Name: cfg.MasterkeySecret.Name}
	secret := &corev1.Secret{}
	err := e.Client.Get(ctx, key, secret)
	if apierrors.IsNotFound(err) && create {
		var random [32]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate master password: %w", err)
		}
		secret = &corev1.Secret{
			Namespace: key.Namespace, Name: key.Name,
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{cfg.MasterkeySecret.Key: []byte(hex.EncodeToString(random[:]))},
		}
		err = e.Client.Create(ctx, secret)
		if apierrors.IsAlreadyExists(err) {
			err = e.Client.Get(ctx, key, secret)
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

func (e *Runtime) createNamespaceIfNotExists(ctx context.Context, namespace string) error {
	err := e.Client.Get(ctx, types.NamespacedName{Name: namespace}, &corev1.Namespace{})
	if err == nil {
		return nil
	} else if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("check if namespace %s exists: %w", namespace, err)
	}

	err = e.Client.Create(ctx, &corev1.Namespace{Name: namespace})
	if err != nil {
		return fmt.Errorf("create namespace %s: %w", namespace, err)
	}

	return nil
}

type Options struct {
	config.HelmfileGlobalConfig
	helmfilePath    string
	environment     string
	envVars         []string
	deployNamespace string
}

func (h Options) commandFor(ctx context.Context, action string) *exec.Cmd {
	helmfilePath := filepath.Join(h.HelmfileUnpackDir, h.helmfilePath)
	cmd := exec.CommandContext(ctx,
		h.HelmfileBin, action,
		"--environment", h.environment,
		"--namespace", h.deployNamespace,
		"--helm-binary", h.HelmBin,
		"--quiet", "--no-color")
	envWithHelmDirs := append(h.envVars,
		fmt.Sprintf("PWD=%s", helmfilePath),
		fmt.Sprintf("HELM_PLUGINS=%s", h.HelmPluginHome),
		fmt.Sprintf("HELM_CACHE_HOME=%s", h.HelmCacheHome),
		fmt.Sprintf("HELM_CONFIG_HOME=%s", h.HelmConfigHome),
		fmt.Sprintf("HELM_DATA_HOME=%s", h.HelmDataHome))
	cmd.Env = append(os.Environ(), envWithHelmDirs...)
	cmd.Dir = helmfilePath
	return cmd
}

func (h Options) Run(ctx context.Context, action string) error { //NOSONAR
	logger := log.FromContext(ctx)
	started := time.Now()
	logger.Info("Running Helmfile command", "action", action, "namespace", h.deployNamespace)
	output, err := h.commandFor(ctx, action).CombinedOutput()
	if err != nil {
		// Command output belongs in bounded diagnostic logs, not conditions or events.
		const maxOutputBytes = 4096
		outputBytes := len(output)
		if outputBytes > maxOutputBytes {
			output = output[outputBytes-maxOutputBytes:]
		}
		logger.Error(err, "Helmfile command failed", "action", action,
			"outputTail", strings.ToValidUTF8(string(output), ""),
			"outputBytes", outputBytes, "outputTruncated", outputBytes > maxOutputBytes,
			"elapsed", time.Since(started).String())
		return fmt.Errorf("%w (see operator logs for output)", err)
	}
	logger.Info("Helmfile command completed", "action", action, "elapsed", time.Since(started).String())
	return nil
}

// UpdateFinalizer manages the Helmfile finalizer. During removal, additional
// finalizers supplied by the deletion step are released in the same atomic patch.
func (e *Runtime) UpdateFinalizer(ctx context.Context, resource *doguv2.Dogu, add bool, additionalFinalizers ...string) error { //NOSONAR
	if add && controllerutil.ContainsFinalizer(resource, Finalizer) && resource.Annotations[ManagedAnnotation] == "true" {
		return nil
	}
	err := retry.OnConflict(func() error {
		var latest doguv2.Dogu
		if err := e.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), &latest); err != nil {
			if !add && apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if latest.UID != resource.UID {
			return fmt.Errorf("dogu was replaced while updating the helmfile finalizer")
		}
		base := latest.DeepCopy()
		if add {
			if !latest.DeletionTimestamp.IsZero() {
				return fmt.Errorf("cannot add helmfile finalizer to a deleting dogu")
			}
			controllerutil.AddFinalizer(&latest, Finalizer)
			if latest.Annotations == nil {
				latest.Annotations = make(map[string]string)
			}
			latest.Annotations[ManagedAnnotation] = "true"
		} else {
			controllerutil.RemoveFinalizer(&latest, Finalizer)
			for _, finalizer := range additionalFinalizers {
				controllerutil.RemoveFinalizer(&latest, finalizer)
			}
		}
		if err := e.Client.Patch(ctx, &latest, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
			return err
		}
		*resource = latest
		return nil
	})
	if err != nil {
		return fmt.Errorf("update helmfile cleanup finalizer: %w", err)
	}
	return nil
}
