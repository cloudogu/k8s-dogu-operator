package deletion

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
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2/helmfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func helmfileDeletionFixture(t *testing.T) (*ExperimentalHelmfileStep, *doguv2.Dogu, string) {
	t.Helper()
	ctx := context.Background()
	step := helmfileTestStep(t)
	step.isEnabled = true
	step.helmfileRuntime.OpenDeskConfig.Namespace = "opendesk"
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "opendesk"}}))
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "password", Namespace: "ecosystem"},
		Data:       map[string][]byte{"masterkey": []byte("existing-master-password")},
	}))
	resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{
		Name: "opendesk", Namespace: "ecosystem",
		Finalizers:  []string{helmfile.Finalizer, finalizerName, "example.com/unrelated"},
		Annotations: map[string]string{helmfile.ManagedAnnotation: "true"},
	}}
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
	require.NoError(t, step.helmfileRuntime.Client.Delete(ctx, resource))
	require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), resource))
	archive := helmfileArchive(t, []tar.Header{{Name: "root/helmfile/helmfile.yaml", Typeflag: tar.TypeReg}}, []string{"releases: []"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)
	step.helmfileRuntime.OpenDeskConfig.HelmfileSource = server.URL
	record := filepath.Join(t.TempDir(), "command")
	step.helmfileRuntime.OpenDeskConfig.ExtraEnvVars = map[string]string{
		"RECORD_FILE": record, "EXTRA": "configured", "MASTER_PASSWORD": "must-not-be-used",
	}
	step.helmfileRuntime.GlobalConfig.HelmfileBin = filepath.Join(t.TempDir(), "helmfile")
	require.NoError(t, os.WriteFile(step.helmfileRuntime.GlobalConfig.HelmfileBin, []byte(
		"#!/bin/sh\nprintf '%s\\n' \"$@\" \"$PWD\" \"$MASTER_PASSWORD\" \"$EXTRA\" > \"$RECORD_FILE\"\n"), 0755))
	return step, resource, record
}

func TestExperimentalHelmfileDeletion(t *testing.T) {
	ctx := context.Background()
	step, resource, record := helmfileDeletionFixture(t)
	// Any write to cluster resources other than the Dogu is a regression.
	step.helmfileRuntime.Client = interceptor.NewClient(step.helmfileRuntime.Client.(client.WithWatch), interceptor.Funcs{
		Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
			t.Fatal("destruction must not create cluster resources")
			return nil
		},
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
			t.Fatal("destruction must use Helmfile, not delete arbitrary cluster resources")
			return nil
		},
	})
	result := step.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	assert.False(t, result.Continue)
	contents, err := os.ReadFile(record)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"destroy", "--environment", "prod", "--namespace", "opendesk", "--helm-binary", "/bin/helm", "--quiet", "--no-color",
		filepath.Join(step.helmfileRuntime.GlobalConfig.HelmfileUnpackDir, "opendesk", "helmfile"), "existing-master-password", "configured",
	}, strings.Split(strings.TrimSpace(string(contents)), "\n"))
	var saved doguv2.Dogu
	require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), &saved))
	assert.Equal(t, []string{"example.com/unrelated"}, saved.Finalizers)
	for _, object := range []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "opendesk"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "domain", Namespace: "ecosystem"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "password", Namespace: "ecosystem"}},
	} {
		require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(object), object))
	}
	assert.True(t, step.Run(ctx, &saved).Completed, "another finalizer must not cause normal Dogu cleanup")
}

func TestExperimentalHelmfileDeletionRetryAfterRestart(t *testing.T) {
	ctx := context.Background()
	step, resource, record := helmfileDeletionFixture(t)
	state := filepath.Join(t.TempDir(), "partial-uninstall")
	step.helmfileRuntime.OpenDeskConfig.ExtraEnvVars["STATE_FILE"] = state
	require.NoError(t, os.WriteFile(step.helmfileRuntime.GlobalConfig.HelmfileBin, []byte(
		"#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$RECORD_FILE\"\nif [ ! -f \"$STATE_FILE\" ]; then touch \"$STATE_FILE\"; printf 'uninstall diagnostic\\n' >&2; exit 1; fi\n"), 0755))
	result := step.Run(ctx, resource)
	require.ErrorContains(t, result.Err, "failed to destroy helmfile")
	assert.NotContains(t, result.Err.Error(), "uninstall diagnostic")
	assert.False(t, result.Completed)
	require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), resource))
	assert.Contains(t, resource.Finalizers, helmfile.Finalizer)
	assert.Contains(t, resource.Finalizers, finalizerName)
	// A new pod has an empty local unpack directory and must reconstruct the Helmfile.
	restarted := *step
	runtimeCopy := *step.helmfileRuntime
	restarted.helmfileRuntime = &runtimeCopy
	restarted.helmfileRuntime.GlobalConfig.HelmfileUnpackDir = t.TempDir()
	result = restarted.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	contents, err := os.ReadFile(record)
	require.NoError(t, err)
	assert.Equal(t, "destroy\ndestroy\n", string(contents))
}

func TestExperimentalHelmfileDeletionRemovesLastFinalizer(t *testing.T) {
	ctx := context.Background()
	step, resource, _ := helmfileDeletionFixture(t)
	base := resource.DeepCopy()
	resource.Finalizers = []string{helmfile.Finalizer, finalizerName}
	require.NoError(t, step.helmfileRuntime.Client.Patch(ctx, resource, client.MergeFrom(base)))
	result := step.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	var saved doguv2.Dogu
	err := step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), &saved)
	assert.True(t, apierrors.IsNotFound(err), "the Dogu should disappear after its last finalizer is removed")
}

func TestExperimentalHelmfileDeletionDoesNotRecreateNamespace(t *testing.T) {
	ctx := context.Background()
	step, resource, _ := helmfileDeletionFixture(t)
	require.NoError(t, step.helmfileRuntime.Client.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "opendesk"}}))
	result := step.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	var namespace corev1.Namespace
	err := step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKey{Name: "opendesk"}, &namespace)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestExperimentalHelmfileDeletionReleasesRemainingDoguFinalizer(t *testing.T) {
	ctx := context.Background()
	step, resource, record := helmfileDeletionFixture(t)
	// An earlier version may have already destroyed the releases and removed
	// only the Helmfile finalizer, leaving the Dogu cleanup finalizer behind.
	base := resource.DeepCopy()
	resource.Finalizers = []string{finalizerName, "example.com/unrelated"}
	require.NoError(t, step.helmfileRuntime.Client.Patch(ctx, resource, client.MergeFrom(base)))
	result := step.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	var saved doguv2.Dogu
	require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), &saved))
	assert.Equal(t, []string{"example.com/unrelated"}, saved.Finalizers)
	_, err := os.Stat(record)
	assert.ErrorIs(t, err, os.ErrNotExist, "completed Helmfile destruction should not run again")
}

func TestExperimentalHelmfileDeletionWithoutDoguFinalizer(t *testing.T) {
	ctx := context.Background()
	step, resource, _ := helmfileDeletionFixture(t)
	base := resource.DeepCopy()
	resource.Finalizers = []string{helmfile.Finalizer, "example.com/unrelated"}
	require.NoError(t, step.helmfileRuntime.Client.Patch(ctx, resource, client.MergeFrom(base)))
	result := step.Run(ctx, resource)
	require.NoError(t, result.Err)
	assert.True(t, result.Completed)
	assert.Equal(t, []string{"example.com/unrelated"}, resource.Finalizers)
}

func TestExperimentalHelmfileDeletionFailuresKeepFinalizer(t *testing.T) {
	for _, scenario := range []string{"disabled", "missing config", "missing domain", "missing masterkey", "download", "finalizer update", "not deleting"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step, resource, _ := helmfileDeletionFixture(t)
			switch scenario {
			case "disabled":
				step.isEnabled = false
			case "missing config":
				step.helmfileRuntime.OpenDeskConfig.Namespace = ""
			case "missing domain":
				step.helmfileRuntime.OpenDeskConfig.DomainConfigMap.Name = "missing"
			case "missing masterkey":
				step.helmfileRuntime.OpenDeskConfig.MasterkeySecret.Name = "missing"
			case "download":
				step.helmfileRuntime.OpenDeskConfig.HelmfileSource = ":invalid"
			case "finalizer update":
				step.helmfileRuntime.Client = interceptor.NewClient(step.helmfileRuntime.Client.(client.WithWatch), interceptor.Funcs{
					Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
						return errors.New("patch failed")
					},
				})
			case "not deleting":
				resource.DeletionTimestamp = nil
			}
			result := step.Run(ctx, resource)
			require.Error(t, result.Err)
			assert.False(t, result.Completed)
			require.NoError(t, step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKeyFromObject(resource), resource))
			assert.Contains(t, resource.Finalizers, helmfile.Finalizer)
			assert.Contains(t, resource.Finalizers, finalizerName)
			if scenario == "missing masterkey" {
				var secret corev1.Secret
				err := step.helmfileRuntime.APIReader.Get(ctx, client.ObjectKey{Namespace: "ecosystem", Name: "missing"}, &secret)
				assert.True(t, apierrors.IsNotFound(err), "destruction must not generate a replacement masterkey")
			}
			if scenario == "finalizer update" {
				step.helmfileRuntime.Client = step.helmfileRuntime.APIReader.(client.WithWatch)
				result = step.Run(ctx, resource)
				require.NoError(t, result.Err)
				assert.True(t, result.Completed)
			}
		})
	}
}

func TestExperimentalHelmfileDeletionSkipsNormalDogus(t *testing.T) {
	step := &ExperimentalHelmfileStep{}
	result := step.Run(context.Background(), &doguv2.Dogu{})
	assert.True(t, result.Continue)
	assert.False(t, result.Completed)
	require.NoError(t, result.Err)
}

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

func helmfileTestStep(t *testing.T) *ExperimentalHelmfileStep {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, doguv2.AddToScheme(scheme))
	step := &ExperimentalHelmfileStep{helmfileRuntime: &helmfile.Runtime{
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
	}}
	step.helmfileRuntime.APIReader = step.helmfileRuntime.Client
	return step
}
