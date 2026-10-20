package helmfile

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func serveHelmfileArchive(t *testing.T, step *Runtime, headers []tar.Header, contents []string) {
	t.Helper()
	data := helmfileArchive(t, headers, contents)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
	t.Cleanup(server.Close)
	step.OpenDeskConfig.HelmfileSource = server.URL
}

func TestNewRuntime(t *testing.T) {
	fixture := helmfileTestStep(t)
	cfg := &config.OperatorConfig{Namespace: fixture.OperatorNamespace, HelmfileGlobalConfig: fixture.GlobalConfig, HelmfileOpenDeskConfig: fixture.OpenDeskConfig}
	actual := NewRuntime(cfg, fixture.Client, fixture.APIReader)
	assert.Equal(t, fixture, actual)
}

func TestPrepareForDestruction(t *testing.T) {
	for _, existingPassword := range []bool{true, false} {
		name := "existing password"
		if !existingPassword {
			name = "missing password"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			serveHelmfileArchive(t, step, []tar.Header{{Name: "root/helmfile.yaml", Typeflag: tar.TypeReg}}, []string{"releases: []"})
			if existingPassword {
				require.NoError(t, step.Client.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "password", Namespace: "ecosystem"}, Data: map[string][]byte{"masterkey": []byte("persisted")}}))
			}
			step.Client = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					t.Fatal("destruction must not create cluster resources")
					return nil
				},
			})
			options, err := step.Prepare(ctx, "opendesk", false)
			if existingPassword {
				require.NoError(t, err)
				assert.Equal(t, []string{"MASTER_PASSWORD=persisted"}, options.envVars)
				assert.Equal(t, "opendesk", options.helmfilePath)
			} else {
				require.ErrorContains(t, err, "read or create master password Secret")
			}
			var namespace corev1.Namespace
			assert.True(t, apierrors.IsNotFound(step.APIReader.Get(ctx, client.ObjectKey{Name: "ecosystem"}, &namespace)))
		})
	}
}

func TestNamespacePreparation(t *testing.T) {
	for _, scenario := range []string{"existing", "create", "get error", "create error"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			failure := errors.New("API unavailable")
			if scenario == "existing" {
				require.NoError(t, step.Client.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "office"}}))
			}
			funcs := interceptor.Funcs{}
			if scenario == "get error" {
				funcs.Get = func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return failure
				}
			}
			if scenario == "create error" {
				funcs.Create = func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return failure }
			}
			step.Client = interceptor.NewClient(step.Client.(client.WithWatch), funcs)
			err := step.createNamespaceIfNotExists(ctx, "office")
			if scenario == "get error" || scenario == "create error" {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
				require.NoError(t, step.APIReader.Get(ctx, client.ObjectKey{Name: "office"}, &corev1.Namespace{}))
			}
		})
	}
}

func TestDownloadHelmfileFailures(t *testing.T) {
	for _, scenario := range []string{"URL", "invalid archive", "missing entrypoint", "unpack directory", "staging directory"} {
		t.Run(scenario, func(t *testing.T) {
			step := helmfileTestStep(t)
			serveHelmfileArchive(t, step, []tar.Header{{Name: "root/values.yaml", Typeflag: tar.TypeReg}}, []string{"{}"})
			switch scenario {
			case "URL":
				step.OpenDeskConfig.HelmfileSource = ":invalid"
			case "invalid archive":
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("invalid")) }))
				t.Cleanup(server.Close)
				step.OpenDeskConfig.HelmfileSource = server.URL
			case "unpack directory":
				step.GlobalConfig.HelmfileUnpackDir = filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(step.GlobalConfig.HelmfileUnpackDir, nil, 0600))
			case "staging directory":
				// The name must be one local path component for MkdirTemp's pattern.
				require.Error(t, step.downloadHelmfile(context.Background(), "invalid/name", logr.Discard()))
				return
			}
			require.Error(t, step.downloadHelmfile(context.Background(), "opendesk", logr.Discard()))
			if scenario != "unpack directory" {
				entries, err := os.ReadDir(step.GlobalConfig.HelmfileUnpackDir)
				require.NoError(t, err)
				assert.Empty(t, entries, "failed downloads must clean up staging directories")
			}
		})
	}
}

func TestExtractHelmfileArchiveAdditionalPaths(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		kind       byte
		want       string
	}{
		{"backslash", `root\escape`, tar.TypeReg, "unsafe archive path"},
		{"empty component", "root//file", tar.TypeReg, "unsafe archive path"},
		{"dot component", "root/./file", tar.TypeReg, "unsafe archive path"},
		{"root file", "root", tar.TypeReg, "root directory"},
		{"hardlink", "root/link", tar.TypeLink, "unsupported archive entry"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := helmfileArchive(t, []tar.Header{{Name: tc.path, Typeflag: tc.kind}}, []string{""})
			require.ErrorContains(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data), t.TempDir(), logr.Discard()), tc.want)
		})
	}
	t.Run("directories and leading dot slash", func(t *testing.T) {
		data := helmfileArchive(t, []tar.Header{{Name: "./root/", Typeflag: tar.TypeDir}, {Name: "./root/sub/", Typeflag: tar.TypeDir}, {Name: "./root/sub/values.yaml", Typeflag: tar.TypeReg}}, []string{"", "", "enabled: true"})
		destination := t.TempDir()
		require.NoError(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data), destination, logr.Discard()))
		actual, err := os.ReadFile(filepath.Join(destination, "sub", "values.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "enabled: true", string(actual))
	})
	for _, kind := range []byte{tar.TypeDir, tar.TypeReg} {
		t.Run(string(kind)+" collision", func(t *testing.T) {
			destination := t.TempDir()
			path := filepath.Join(destination, "blocked")
			if kind == tar.TypeDir {
				require.NoError(t, os.WriteFile(path, nil, 0600))
			} else {
				require.NoError(t, os.Mkdir(path, 0700))
			}
			data := helmfileArchive(t, []tar.Header{{Name: "root/blocked", Typeflag: kind}}, []string{""})
			require.Error(t, extractHelmfileArchive(context.Background(), bytes.NewReader(data), destination, logr.Discard()))
		})
	}
}

func TestHelmfileWorkingDir(t *testing.T) {
	for _, directory := range []string{".", "helmfile"} {
		for _, name := range []string{"helmfile.yaml", "helmfile.yml", "helmfile.yaml.gotmpl", "helmfile.yml.gotmpl"} {
			t.Run(filepath.Join(directory, name), func(t *testing.T) {
				root := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(root, directory), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(root, directory, name), nil, 0600))
				actual, err := helmfileWorkingDir(root)
				require.NoError(t, err)
				assert.Equal(t, directory, actual)
			})
		}
	}
	t.Run("entrypoint is a directory", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(root, "helmfile.yaml"), 0700))
		_, err := helmfileWorkingDir(root)
		require.ErrorContains(t, err, "no helmfile entrypoint")
	})
	t.Run("base is a file", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(root, nil, 0600))
		_, err := helmfileWorkingDir(root)
		require.Error(t, err)
	})
}

func TestHelmfileFinalizerEdgeCases(t *testing.T) {
	for _, scenario := range []string{"already managed", "remove", "missing removal", "missing addition", "replaced", "deleting", "read failure", "patch failure"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Namespace: "ecosystem", UID: "original", Finalizers: []string{Finalizer, "example.com/cleanup", "example.com/keep"}}}
			require.NoError(t, step.Client.Create(ctx, resource))
			add := true
			failure := errors.New("API unavailable")
			switch scenario {
			case "already managed":
				resource.Annotations = map[string]string{ManagedAnnotation: "true"}
				step.APIReader = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					t.Fatal("already managed resources need no API read")
					return nil
				}})
			case "remove":
				add = false
			case "missing removal", "missing addition":
				require.NoError(t, step.Client.Delete(ctx, resource))
				require.NoError(t, step.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), resource))
				resource.Finalizers = nil
				require.NoError(t, step.Client.Update(ctx, resource))
				add = scenario == "missing addition"
			case "replaced":
				resource.UID = "replacement"
			case "deleting":
				require.NoError(t, step.Client.Delete(ctx, resource))
			case "read failure":
				step.APIReader = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return failure
				}})
			case "patch failure":
				step.Client = interceptor.NewClient(step.Client.(client.WithWatch), interceptor.Funcs{Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
					return failure
				}})
			}
			err := step.UpdateFinalizer(ctx, resource, add, "example.com/cleanup")
			switch scenario {
			case "already managed", "missing removal":
				require.NoError(t, err)
			case "remove":
				require.NoError(t, err)
				var saved doguv2.Dogu
				require.NoError(t, step.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), &saved))
				assert.Equal(t, []string{"example.com/keep"}, saved.Finalizers)
				assert.Equal(t, saved.Finalizers, resource.Finalizers)
			case "read failure", "patch failure":
				require.ErrorIs(t, err, failure)
			case "replaced":
				require.ErrorContains(t, err, "replaced")
			case "deleting":
				require.ErrorContains(t, err, "deleting")
			default:
				require.Error(t, err)
			}
		})
	}
}
