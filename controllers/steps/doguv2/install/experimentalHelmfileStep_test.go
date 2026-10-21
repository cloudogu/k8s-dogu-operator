package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2/helmfile"
	reg "github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func helmfileTestStep(t *testing.T) *ExperimentalHelmfileStep {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
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
	configMaps := k8sfake.NewClientset().CoreV1().ConfigMaps("ecosystem")
	repository := reg.NewLocalDoguDescriptorRepository(configMaps)
	versions := reg.NewDoguVersionRegistry(configMaps)
	step.localDoguFetcher = cesregistry.NewLocalDoguFetcher(versions, repository)
	step.doguRegistrator = cesregistry.NewCESDoguRegistrator(versions, repository)
	step.descriptorOwnerReferenceStep = NewOwnerReferenceStep(repository, scheme)
	return step
}

func TestExperimentalHelmfileRunCapturesFailureOutput(t *testing.T) {
	var logs strings.Builder
	ctx := log.IntoContext(context.Background(), funcr.New(func(_, message string) {
		logs.WriteString(message)
	}, funcr.Options{}))
	step := helmfileTestStep(t)
	step.isEnabled = true
	resource := &doguv2.Dogu{Name: "opendesk"}
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
	fetcher := newMockResourceDoguFetcher(t)
	fetcher.EXPECT().FetchWithResource(ctx, resource).Return(&core.Dogu{
		Name: "internal/opendesk", Version: "1.18.2-1",
		Properties: map[string]string{"kind": "Helmfile"},
	}, nil, nil)
	step.doguFetcher = fetcher

	doguClient := newMockDoguInterface(t)
	doguClient.EXPECT().UpdateStatusWithRetry(ctx, resource, mock.Anything, metav1.UpdateOptions{}).
		RunAndReturn(func(_ context.Context, dogu *doguv2.Dogu, modify func(doguv2.DoguStatus) doguv2.DoguStatus, _ metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updated := dogu.DeepCopy()
			updated.Status = modify(updated.Status)
			assert.Equal(t, doguv2.DoguStatusInstalling, updated.Status.Status)
			return updated, nil
		}).Once()
	step.doguInterface = doguClient

	archive := helmfileArchive(t, []tar.Header{
		{Name: "opendesk/helmfile.yaml", Typeflag: tar.TypeReg},
	}, []string{"releases: []"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, doguv2.DoguStatusInstalling, resource.Status.Status, "status must be set before preparing and executing Helmfile")
		_, err := w.Write(archive)
		assert.NoError(t, err)
	}))
	defer server.Close()
	step.helmfileRuntime.OpenDeskConfig.HelmfileSource = server.URL
	step.helmfileRuntime.GlobalConfig.HelmfileBin = filepath.Join(t.TempDir(), "helmfile")
	require.NoError(t, os.WriteFile(step.helmfileRuntime.GlobalConfig.HelmfileBin, []byte(
		"#!/bin/sh\ni=0\nwhile [ \"$i\" -lt 4000 ]; do printf 'comparison noise\\n'; i=$((i + 1)); done\nprintf 'stdout diagnostic\\n'\nprintf 'stderr diagnostic\\n' >&2\nexit 1\n",
	), 0755))

	result := step.Run(ctx, resource)
	require.Error(t, result.Err)
	assert.Contains(t, resource.Finalizers, helmfile.Finalizer, "failed installations still need cleanup")
	assert.False(t, result.Continue)
	assert.ErrorContains(t, result.Err, "failed to apply helmfile of opendesk")
	assert.NotContains(t, result.Err.Error(), "stdout diagnostic")
	assert.NotContains(t, result.Err.Error(), "stderr diagnostic")
	assert.NotContains(t, result.Err.Error(), "comparison noise")
	assert.Less(t, len(result.Err.Error()), 1024)
	assert.Contains(t, logs.String(), "stdout diagnostic")
	assert.Contains(t, logs.String(), "stderr diagnostic")
	assert.Contains(t, logs.String(), `"outputTruncated"=true`)
	assert.Less(t, logs.Len(), 8192)
	var exitError *exec.ExitError
	require.ErrorAs(t, result.Err, &exitError)
	assert.Equal(t, 1, exitError.ExitCode())
	assert.Equal(t, doguv2.DoguStatusInstalling, resource.Status.Status)
}

func TestExperimentalHelmfileInstallingStatusFailurePreventsApply(t *testing.T) {
	ctx := context.Background()
	step := helmfileTestStep(t)
	step.isEnabled = true
	resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Namespace: "ecosystem"}}
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
	fetcher := newMockResourceDoguFetcher(t)
	fetcher.EXPECT().FetchWithResource(ctx, resource).Return(&core.Dogu{
		Name: "internal/opendesk", Version: "1.18.2-1", Properties: map[string]string{"kind": "Helmfile"},
	}, nil, nil)
	step.doguFetcher = fetcher
	failure := errors.New("status update failed")
	doguClient := newMockDoguInterface(t)
	doguClient.EXPECT().UpdateStatusWithRetry(ctx, resource, mock.Anything, metav1.UpdateOptions{}).Return(nil, failure).Once()
	step.doguInterface = doguClient

	// No Helmfile source is configured. Preparing or executing Helmfile would yield a different error.
	result := step.Run(ctx, resource)
	require.ErrorIs(t, result.Err, failure)
	assert.ErrorContains(t, result.Err, "failed to set helmfile dogu status to installing")
	assert.False(t, result.Completed)
	assert.False(t, result.Continue)
}

func TestExperimentalHelmfileFinalizerFailurePreventsDownload(t *testing.T) {
	ctx := context.Background()
	step := helmfileTestStep(t)
	step.isEnabled = true
	resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk"}}
	require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
	fetcher := newMockResourceDoguFetcher(t)
	fetcher.EXPECT().FetchWithResource(ctx, resource).Return(&core.Dogu{Properties: map[string]string{"kind": "Helmfile"}}, nil, nil)
	step.doguFetcher = fetcher
	failure := errors.New("unable to persist finalizer")
	step.helmfileRuntime.Client = interceptor.NewClient(step.helmfileRuntime.Client.(client.WithWatch), interceptor.Funcs{
		Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
			return failure
		},
	})
	// No Helmfile source is configured. A download attempt would return a different error.
	result := step.Run(ctx, resource)
	require.ErrorIs(t, result.Err, failure)
	assert.False(t, result.Completed)
}

func TestExperimentalHelmfileRegistersAppliedDogu(t *testing.T) {
	for _, previousVersion := range []string{"", "1.18.1-1", "1.18.2-1"} {
		t.Run("previous version "+previousVersion, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			step.isEnabled = true
			resource := &doguv2.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Namespace: "ecosystem", UID: "opendesk-uid", Generation: 1},
				Spec:       doguv2.DoguSpec{Name: "internal/opendesk", Version: "1.18.2-1"},
				Status: doguv2.DoguStatus{Conditions: []metav1.Condition{{
					Type: conditionHelmfileApplied, Status: metav1.ConditionTrue, ObservedGeneration: 1,
				}}},
			}
			require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
			descriptor := &core.Dogu{Name: resource.Spec.Name, Version: resource.Spec.Version, Properties: map[string]string{"kind": "Helmfile"}}
			fetcher := newMockResourceDoguFetcher(t)
			fetcher.EXPECT().FetchWithResource(ctx, resource).Return(descriptor, nil, nil)
			step.doguFetcher = fetcher

			clientset := k8sfake.NewClientset()
			configMaps := clientset.CoreV1().ConfigMaps(resource.Namespace)
			repository := reg.NewLocalDoguDescriptorRepository(configMaps)
			versions := reg.NewDoguVersionRegistry(configMaps)
			registrator := cesregistry.NewCESDoguRegistrator(versions, repository)
			if previousVersion != "" {
				previous := *descriptor
				previous.Version = previousVersion
				require.NoError(t, registrator.RegisterNewDogu(ctx, resource, &previous))
			}
			step.localDoguFetcher = cesregistry.NewLocalDoguFetcher(versions, repository)
			step.doguRegistrator = registrator
			step.descriptorOwnerReferenceStep = NewOwnerReferenceStep(repository, step.helmfileRuntime.Client.Scheme())

			// No Helmfile source is configured: an unintended apply would fail.
			for range 2 {
				result := step.Run(ctx, resource)
				require.NoError(t, result.Err)
				assert.Equal(t, 10*time.Second, result.RequeueAfter, "registration must not bypass readiness")
				assert.False(t, result.Continue, "normal Dogu installation must not run")
				assert.False(t, result.Completed)

				cm, err := configMaps.Get(ctx, "dogu-spec-opendesk", metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, descriptor.Version, cm.Data["current"])
				var registered core.Dogu
				require.NoError(t, json.Unmarshal([]byte(cm.Data[descriptor.Version]), &registered))
				assert.Equal(t, *descriptor, registered)
				require.Len(t, cm.OwnerReferences, 1)
				assert.Equal(t, resource.UID, cm.OwnerReferences[0].UID)
				assert.Equal(t, resource.Name, cm.OwnerReferences[0].Name)
				assert.Equal(t, "k8s.cloudogu.com/v2", cm.OwnerReferences[0].APIVersion)
				assert.Equal(t, "Dogu", cm.OwnerReferences[0].Kind)
				assert.True(t, *cm.OwnerReferences[0].Controller)
				assert.True(t, *cm.OwnerReferences[0].BlockOwnerDeletion)
				current, err := versions.GetCurrentOfAll(ctx)
				require.NoError(t, err)
				require.Len(t, current, 1, "Admin's backend must discover the registration")
				assert.Equal(t, cescommons.SimpleName("opendesk"), current[0].Name)
				assert.Equal(t, descriptor.Version, current[0].Version.Raw)
			}

			// Ready Helmfiles still terminate the use case instead of running normal Dogu steps.
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "office", Namespace: resource.Namespace},
				Status:     appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1},
			}
			require.NoError(t, step.helmfileRuntime.Client.Create(ctx, deployment))
			doguClient := newMockDoguInterface(t)
			doguClient.EXPECT().UpdateStatusWithRetry(ctx, resource, mock.Anything, metav1.UpdateOptions{}).
				RunAndReturn(func(_ context.Context, dogu *doguv2.Dogu, modify func(doguv2.DoguStatus) doguv2.DoguStatus, _ metav1.UpdateOptions) (*doguv2.Dogu, error) {
					updated := dogu.DeepCopy()
					updated.Status = modify(updated.Status)
					return updated, nil
				})
			step.doguInterface = doguClient
			assert.Equal(t, steps.Complete(), step.Run(ctx, resource))
			assert.Equal(t, doguv2.DoguStatusInstalled, resource.Status.Status)
		})
	}
}

func TestExperimentalHelmfileRegistrationFailurePreventsApply(t *testing.T) {
	for _, failureStage := range []string{"read registry", "write descriptor", "enable version", "set owner reference"} {
		t.Run(failureStage, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			step.isEnabled = true
			resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Namespace: "ecosystem"}}
			require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
			descriptor := &core.Dogu{Name: "internal/opendesk", Version: "1.18.2-1", Properties: map[string]string{"kind": "Helmfile"}}
			fetcher := newMockResourceDoguFetcher(t)
			fetcher.EXPECT().FetchWithResource(ctx, resource).Return(descriptor, nil, nil)
			step.doguFetcher = fetcher
			clientset := k8sfake.NewClientset()
			configMaps := clientset.CoreV1().ConfigMaps(resource.Namespace)
			repository := reg.NewLocalDoguDescriptorRepository(configMaps)
			versions := reg.NewDoguVersionRegistry(configMaps)
			step.localDoguFetcher = cesregistry.NewLocalDoguFetcher(versions, repository)
			step.doguRegistrator = cesregistry.NewCESDoguRegistrator(versions, repository)
			owner := NewMockLocalDoguDescriptorOwnerReferenceStep(t)
			step.descriptorOwnerReferenceStep = owner
			failure := errors.New("registry unavailable")
			switch failureStage {
			case "set owner reference":
				owner.EXPECT().Run(ctx, resource).Return(steps.RequeueWithError(failure))
			default:
				verb := "update"
				if failureStage == "read registry" {
					verb = "get"
				}
				clientset.PrependReactor(verb, "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
					if failureStage == "enable version" && action.(k8stesting.UpdateAction).GetObject().(*corev1.ConfigMap).Data["current"] == "" {
						return false, nil, nil
					}
					return true, nil, failure
				})
			}
			// No source is configured. If apply were attempted, a different error would result.
			result := step.Run(ctx, resource)
			require.ErrorIs(t, result.Err, failure)
			assert.False(t, result.Continue)
			assert.False(t, result.Completed)
			assert.Contains(t, resource.Finalizers, helmfile.Finalizer)
		})
	}
}
