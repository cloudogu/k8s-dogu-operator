package helmfile

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

func helmfileArchive(t *testing.T, headers []tar.Header, contents []string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	for i, header := range headers {
		header.Size = int64(len(contents[i]))
		if header.Typeflag != tar.TypeXGlobalHeader {
			header.Mode = 0644
		}
		require.NoError(t, tw.WriteHeader(&header))
		_, err := tw.Write([]byte(contents[i]))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buffer.Bytes()
}

func helmfileTestStep(t *testing.T) *Runtime {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, doguv2.AddToScheme(scheme))
	step := &Runtime{
		OperatorNamespace: "ecosystem",
		GlobalConfig: config.HelmfileGlobalConfig{
			HelmfileUnpackDir: t.TempDir(), HelmfileBin: "/bin/helmfile", HelmBin: "/bin/helm",
			HelmPluginHome: "/plugins", HelmCacheHome: "/cache", HelmConfigHome: "/config", HelmDataHome: "/data",
		},
		OpenDeskConfig: config.HelmfileOpenDeskConfig{
			Namespace: "ecosystem", Environment: "prod", ValuesFilePath: "helmfile/environments/prod/values.yaml.gotmpl",
			DomainConfigMap: config.K8sConfigRef{Name: "domain", Key: "domain"},
			MasterkeySecret: config.K8sConfigRef{Name: "password", Key: "masterkey"},
		},
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ConfigMap{
			Name: "domain", Namespace: "ecosystem",
			Data: map[string]string{"domain": "office.example.org"},
		}).Build(),
	}
	step.APIReader = step.Client
	return step
}

func TestExperimentalHelmfilePreparation(t *testing.T) {
	ctx := context.Background()
	step := helmfileTestStep(t)
	archive := helmfileArchive(t, []tar.Header{
		{Name: "opendesk-v1/helmfile/helmfile.yaml.gotmpl", Typeflag: tar.TypeReg},
		{Name: "opendesk-v1/helmfile/environments/prod/values.yaml.gotmpl", Typeflag: tar.TypeReg},
	}, []string{"releases: []", "{{ old template }}"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(archive)
		assert.NoError(t, err)
	}))
	defer server.Close()
	step.OpenDeskConfig.HelmfileSource = server.URL
	step.OpenDeskConfig.ExtraValues = map[string]any{
		"global": map[string]any{"domain": "ignored", "other": "retained"},
		"nested": map[string]any{"enabled": true},
	}
	step.OpenDeskConfig.ExtraEnvVars = map[string]string{"MASTER_PASSWORD": "ignored", "EXTRA": "value"}
	t.Setenv("MASTER_PASSWORD", "inherited")
	options, err := step.Prepare(ctx, "opendesk", true)
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(step.GlobalConfig.HelmfileUnpackDir, "opendesk", step.OpenDeskConfig.ValuesFilePath))
	require.NoError(t, err)
	var values map[string]any
	require.NoError(t, yaml.Unmarshal(data, &values))
	assert.Equal(t, map[string]any{"domain": "office.example.org", "other": "retained"}, values["global"])
	assert.Equal(t, map[string]any{"enabled": true}, values["nested"])
	assert.Equal(t, "ignored", step.OpenDeskConfig.ExtraValues["global"].(map[string]any)["domain"])
	var secret corev1.Secret
	require.NoError(t, step.Client.Get(ctx, client.ObjectKey{Namespace: "ecosystem", Name: "password"}, &secret))
	password := string(secret.Data["masterkey"])
	assert.Len(t, password, 64)
	assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
	assert.Equal(t, []string{"EXTRA=value", "MASTER_PASSWORD=" + password}, options.envVars)
	cmd := options.commandFor(ctx, "apply")
	assert.Equal(t, filepath.Join(step.GlobalConfig.HelmfileUnpackDir, "opendesk", "helmfile"), cmd.Dir)
	assert.Equal(t, []string{"/bin/helmfile", "apply", "--environment", "prod", "--namespace", "ecosystem", "--helm-binary", "/bin/helm", "--quiet", "--no-color"}, cmd.Args)
	assert.Contains(t, cmd.Env, "HELM_PLUGINS=/plugins")
	assert.Contains(t, cmd.Env, "HELM_CACHE_HOME=/cache")
	assert.Contains(t, cmd.Env, "HELM_CONFIG_HOME=/config")
	assert.Contains(t, cmd.Env, "HELM_DATA_HOME=/data")
	assert.Greater(t, strings.LastIndex(strings.Join(cmd.Env, "\n"), "MASTER_PASSWORD="+password), strings.Index(strings.Join(cmd.Env, "\n"), "MASTER_PASSWORD=inherited"))
	// Re-extraction must remove stale files and preserve the persisted password.
	require.NoError(t, os.WriteFile(filepath.Join(cmd.Dir, "stale"), []byte("old"), 0644))
	second, err := step.Prepare(ctx, "opendesk", true)
	require.NoError(t, err)
	assert.Equal(t, options.envVars, second.envVars)
	_, err = os.Stat(filepath.Join(cmd.Dir, "stale"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestExtractHelmfileArchive(t *testing.T) {
	for _, position := range []int{0, 1} {
		name := "global PAX header before files"
		if position == 1 {
			name = "global PAX header between files"
		}
		t.Run(name, func(t *testing.T) {
			globalHeader := tar.Header{
				Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader,
				Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": "source revision"},
			}
			headers := []tar.Header{
				{Name: "root/helmfile.yaml", Typeflag: tar.TypeReg},
				{Name: "root/values.yaml", Typeflag: tar.TypeReg},
			}
			contents := []string{"releases: []", "global: {}"}
			if position == 0 {
				headers = append([]tar.Header{globalHeader}, headers...)
				contents = append([]string{""}, contents...)
			} else {
				headers = []tar.Header{headers[0], globalHeader, headers[1]}
				contents = []string{contents[0], "", contents[1]}
			}
			destination := t.TempDir()
			require.NoError(t, extractHelmfileArchive(context.Background(), bytes.NewReader(helmfileArchive(t, headers, contents)), destination, logr.Discard()))
			for path, expected := range map[string]string{"helmfile.yaml": "releases: []", "values.yaml": "global: {}"} {
				actual, err := os.ReadFile(filepath.Join(destination, path))
				require.NoError(t, err)
				assert.Equal(t, expected, string(actual))
			}
			_, err := os.Stat(filepath.Join(destination, "pax_global_header"))
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
	for _, tc := range []struct {
		name     string
		headers  []tar.Header
		contents []string
	}{
		{"traversal", []tar.Header{{Name: "root/../escape", Typeflag: tar.TypeReg}}, []string{"bad"}},
		{"absolute", []tar.Header{{Name: "/escape", Typeflag: tar.TypeReg}}, []string{"bad"}},
		{"symlink", []tar.Header{{Name: "root/link", Typeflag: tar.TypeSymlink, Linkname: "/tmp"}}, []string{""}},
		{"multiple roots", []tar.Header{{Name: "one/a", Typeflag: tar.TypeReg}, {Name: "two/b", Typeflag: tar.TypeReg}}, []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, extractHelmfileArchive(context.Background(), bytes.NewReader(helmfileArchive(t, tc.headers, tc.contents)), t.TempDir(), logr.Discard()))
		})
	}
	data := helmfileArchive(t, []tar.Header{{Name: "root/helmfile.yaml", Typeflag: tar.TypeReg}}, []string{"releases: []"})
	t.Run("invalid gzip", func(t *testing.T) {
		assert.Error(t, extractHelmfileArchive(context.Background(), strings.NewReader("invalid"), t.TempDir(), logr.Discard()))
	})
	t.Run("truncated gzip", func(t *testing.T) {
		assert.Error(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data[:len(data)-5]), t.TempDir(), logr.Discard()))
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.ErrorIs(t, extractHelmfileArchive(ctx, bytes.NewReader(data), t.TempDir(), logr.Discard()), context.Canceled)
	})
	t.Run("filesystem failure", func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(destination, nil, 0644))
		assert.Error(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data), destination, logr.Discard()))
	})
	t.Run("root preferred", func(t *testing.T) {
		destination := t.TempDir()
		require.NoError(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data), destination, logr.Discard()))
		directory, err := helmfileWorkingDir(destination)
		require.NoError(t, err)
		assert.Equal(t, ".", directory)
	})
}

func TestHelmfileMasterPassword(t *testing.T) {
	ctx := context.Background()
	for _, value := range []string{"persisted", ""} {
		t.Run("existing "+value, func(t *testing.T) {
			step := helmfileTestStep(t)
			require.NoError(t, step.Client.Create(ctx, &corev1.Secret{
				Name: "password", Namespace: "ecosystem", Data: map[string][]byte{"masterkey": []byte(value)},
			}))
			password, err := step.getMasterPassword(ctx, true)
			if value == "" {
				assert.ErrorContains(t, err, "missing or empty key")
			} else {
				require.NoError(t, err)
				assert.Equal(t, value, password)
			}
		})
	}
	t.Run("creation race", func(t *testing.T) {
		step := helmfileTestStep(t)
		step.Client = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				winner := obj.(*corev1.Secret).DeepCopy()
				winner.Data["masterkey"] = []byte("winner")
				require.NoError(t, c.Create(ctx, winner, opts...))
				return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, obj.GetName())
			},
		})
		password, err := step.getMasterPassword(ctx, true)
		require.NoError(t, err)
		assert.Equal(t, "winner", password)
	})
	for _, operation := range []string{"get", "create"} {
		t.Run(operation+" failure", func(t *testing.T) {
			step := helmfileTestStep(t)
			failure := errors.New("API unavailable")
			funcs := interceptor.Funcs{}
			if operation == "get" {
				funcs.Get = func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return failure
				}
			} else {
				funcs.Create = func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return failure }
			}
			step.Client = interceptor.NewClient(step.Client.(client.WithWatch), funcs)
			_, err := step.getMasterPassword(ctx, true)
			assert.ErrorIs(t, err, failure)
		})
	}
}

func TestHelmfilePreparationFailures(t *testing.T) {
	archive := helmfileArchive(t, []tar.Header{{Name: "root/helmfile.yaml", Typeflag: tar.TypeReg}}, []string{"releases: []"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failure" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	for _, scenario := range []string{"HTTP", "missing ConfigMap", "empty domain", "global type", "values path", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			step := helmfileTestStep(t)
			step.OpenDeskConfig.HelmfileSource = server.URL
			ctx := context.Background()
			switch scenario {
			case "HTTP":
				step.OpenDeskConfig.HelmfileSource += "/failure"
			case "missing ConfigMap":
				step.OpenDeskConfig.DomainConfigMap.Name = "missing"
			case "empty domain":
				step.OpenDeskConfig.DomainConfigMap.Key = "missing"
			case "global type":
				step.OpenDeskConfig.ExtraValues = map[string]any{"global": "invalid"}
			case "values path":
				step.OpenDeskConfig.ValuesFilePath = "../escape"
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, err := step.Prepare(ctx, "opendesk", true)
			assert.Error(t, err)
		})
	}
}
func TestExperimentalHelmfileFinalizer(t *testing.T) {
	ctx := context.Background()
	step := helmfileTestStep(t)
	resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{
		Name: "opendesk", Namespace: "ecosystem", Finalizers: []string{"example.com/other"},
	}}
	require.NoError(t, step.Client.Create(ctx, resource))
	patches := 0
	step.Client = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			patches++
			if patches == 1 {
				var concurrent doguv2.Dogu
				require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(resource), &concurrent))
				concurrent.Finalizers = append(concurrent.Finalizers, "example.com/concurrent")
				require.NoError(t, c.Update(ctx, &concurrent))
				return apierrors.NewConflict(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogus"}, resource.Name, errors.New("concurrent update"))
			}
			return c.Patch(ctx, obj, patch, opts...)
		},
	})
	require.NoError(t, step.UpdateFinalizer(ctx, resource, true))
	assert.Equal(t, 2, patches)
	assert.ElementsMatch(t, []string{"example.com/other", "example.com/concurrent", Finalizer}, resource.Finalizers)
	assert.Equal(t, "true", resource.Annotations[ManagedAnnotation])
}
