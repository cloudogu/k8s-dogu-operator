package install

import (
	"archive/tar"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestHelmfileInstallationGuards(t *testing.T) {
	for _, scenario := range []string{"disabled", "ordinary dogu", "descriptor error", "unsupported dogu", "deleting"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			step.isEnabled = scenario != "disabled"
			resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk"}}
			descriptor := &core.Dogu{Properties: map[string]string{"kind": "Helmfile"}}
			failure := errors.New("descriptor unavailable")
			var fetchErr error
			switch scenario {
			case "ordinary dogu":
				descriptor.Properties = nil
			case "descriptor error":
				fetchErr = failure
			case "unsupported dogu":
				resource.Name = "unsupported"
			case "deleting":
				now := metav1.Now()
				resource.DeletionTimestamp = &now
			}
			if step.isEnabled {
				fetcher := newMockResourceDoguFetcher(t)
				fetcher.EXPECT().FetchWithResource(ctx, resource).Return(descriptor, nil, fetchErr)
				step.doguFetcher = fetcher
			}
			result := step.Run(ctx, resource)
			if scenario == "disabled" || scenario == "ordinary dogu" {
				assert.Equal(t, steps.Continue(), result)
			} else {
				require.Error(t, result.Err)
				assert.False(t, result.Continue)
				assert.False(t, result.Completed)
				if fetchErr != nil {
					assert.ErrorIs(t, result.Err, failure)
				}
			}
		})
	}
}

func TestHelmfileReadiness(t *testing.T) {
	for _, scenario := range []string{"empty namespace", "list error", "no deployments", "deleting", "old generation", "replicas", "updated", "ready", "available", "zero replicas", "status error", "ready default replicas"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Generation: 7}, Spec: doguv2.DoguSpec{Version: "1.18.2-1"}}
			failure := errors.New("API unavailable")
			replicas := int32(1)
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "office", Namespace: "ecosystem", Generation: 2},
				Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
				Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1},
			}
			switch scenario {
			case "empty namespace":
				step.helmfileRuntime.OpenDeskConfig.Namespace = ""
			case "list error":
				step.helmfileRuntime.APIReader = interceptor.NewClient(step.helmfileRuntime.Client.(client.WithWatch), interceptor.Funcs{List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error { return failure }})
			case "deleting":
				deployment.Finalizers = []string{"example.com/cleanup"}
			case "old generation":
				deployment.Status.ObservedGeneration = 1
			case "replicas":
				deployment.Status.Replicas = 0
			case "updated":
				deployment.Status.UpdatedReplicas = 0
			case "ready":
				deployment.Status.ReadyReplicas = 0
			case "available":
				deployment.Status.AvailableReplicas = 0
			case "zero replicas":
				replicas = 0
				deployment.Status = appsv1.DeploymentStatus{ObservedGeneration: 2}
			case "ready default replicas":
				deployment.Spec.Replicas = nil
			}
			if scenario != "no deployments" {
				require.NoError(t, step.helmfileRuntime.Client.Create(ctx, deployment))
				if scenario == "deleting" {
					require.NoError(t, step.helmfileRuntime.Client.Delete(ctx, deployment))
				}
			}
			if scenario == "zero replicas" || scenario == "ready default replicas" || scenario == "status error" {
				doguClient := newMockDoguInterface(t)
				doguClient.EXPECT().UpdateStatusWithRetry(ctx, resource, mock.Anything, metav1.UpdateOptions{}).
					RunAndReturn(func(_ context.Context, dogu *doguv2.Dogu, modify func(doguv2.DoguStatus) doguv2.DoguStatus, _ metav1.UpdateOptions) (*doguv2.Dogu, error) {
						if scenario == "status error" {
							return nil, failure
						}
						updated := dogu.DeepCopy()
						updated.Status = modify(updated.Status)
						return updated, nil
					})
				step.doguInterface = doguClient
			}
			result := step.checkHelmfileDeploymentsReady(ctx, resource)
			switch scenario {
			case "empty namespace":
				require.ErrorContains(t, result.Err, "requires a namespace")
			case "list error", "status error":
				require.ErrorIs(t, result.Err, failure)
			case "zero replicas", "ready default replicas":
				assert.Equal(t, steps.Continue(), result)
				assert.Equal(t, doguv2.DoguStatusInstalled, resource.Status.Status)
				assert.Equal(t, resource.Spec.Version, resource.Status.InstalledVersion)
				assert.Equal(t, doguv2.AvailableHealthStatus, resource.Status.Health)
				for _, kind := range []string{doguv2.ConditionReady, doguv2.ConditionHealthy} {
					condition := meta.FindStatusCondition(resource.Status.Conditions, kind)
					require.NotNil(t, condition)
					assert.Equal(t, metav1.ConditionTrue, condition.Status)
					assert.Equal(t, resource.Generation, condition.ObservedGeneration)
				}
			default:
				assert.Equal(t, steps.RequeueAfter(10*time.Second), result)
			}
		})
	}
}

func TestHelmfileSuccessfulApply(t *testing.T) {
	for _, scenario := range []string{"success", "generation changes", "applied status error", "preparation error"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			step := helmfileTestStep(t)
			step.isEnabled = true
			resource := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "opendesk", Namespace: "ecosystem", Generation: 1}, Spec: doguv2.DoguSpec{Name: "internal/opendesk", Version: "1.18.2-1"}}
			require.NoError(t, step.helmfileRuntime.Client.Create(ctx, resource))
			descriptor := &core.Dogu{Name: resource.Spec.Name, Version: resource.Spec.Version, Properties: map[string]string{"kind": "Helmfile"}}
			fetcher := newMockResourceDoguFetcher(t)
			fetcher.EXPECT().FetchWithResource(ctx, resource).Return(descriptor, nil, nil)
			step.doguFetcher = fetcher
			data := helmfileArchive(t, []tar.Header{{Name: "root/helmfile.yaml", Typeflag: tar.TypeReg}}, []string{"releases: []"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
			t.Cleanup(server.Close)
			step.helmfileRuntime.OpenDeskConfig.HelmfileSource = server.URL
			if scenario == "preparation error" {
				step.helmfileRuntime.OpenDeskConfig.ValuesFilePath = "../escape"
			}
			record := filepath.Join(t.TempDir(), "calls")
			step.helmfileRuntime.OpenDeskConfig.ExtraEnvVars = map[string]string{"RECORD": record}
			binary := filepath.Join(t.TempDir(), "helmfile")
			require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$RECORD\"\n"), 0700))
			step.helmfileRuntime.GlobalConfig.HelmfileBin = binary
			failure := errors.New("status unavailable")
			doguClient := newMockDoguInterface(t)
			calls := 0
			doguClient.EXPECT().UpdateStatusWithRetry(ctx, resource, mock.Anything, metav1.UpdateOptions{}).
				RunAndReturn(func(_ context.Context, dogu *doguv2.Dogu, modify func(doguv2.DoguStatus) doguv2.DoguStatus, _ metav1.UpdateOptions) (*doguv2.Dogu, error) {
					calls++
					if calls == 2 && scenario == "applied status error" {
						return nil, failure
					}
					updated := dogu.DeepCopy()
					updated.Status = modify(updated.Status)
					if calls == 2 && scenario == "generation changes" {
						updated.Generation++
					}
					return updated, nil
				})
			step.doguInterface = doguClient
			result := step.Run(ctx, resource)
			if scenario == "preparation error" {
				require.ErrorContains(t, result.Err, "failed to extract and configure helmfile")
				_, err := os.Stat(record)
				assert.ErrorIs(t, err, os.ErrNotExist)
				assert.Equal(t, 1, calls)
				return
			}
			actual, err := os.ReadFile(record)
			require.NoError(t, err)
			assert.Equal(t, "apply\n", string(actual))
			assert.Equal(t, 2, calls)
			if scenario == "applied status error" {
				require.ErrorIs(t, result.Err, failure)
				return
			}
			assert.Equal(t, steps.RequeueAfter(10*time.Second), result)
			applied := meta.FindStatusCondition(resource.Status.Conditions, conditionHelmfileApplied)
			require.NotNil(t, applied)
			assert.Equal(t, metav1.ConditionTrue, applied.Status)
			assert.Equal(t, int64(1), applied.ObservedGeneration)
			if scenario == "success" {
				// A second reconciliation waits for deployments without repeating apply.
				assert.Equal(t, steps.RequeueAfter(10*time.Second), step.Run(ctx, resource))
				actual, err = os.ReadFile(record)
				require.NoError(t, err)
				assert.Equal(t, "apply\n", string(actual))
				assert.Equal(t, 2, calls)
			}
		})
	}
}
