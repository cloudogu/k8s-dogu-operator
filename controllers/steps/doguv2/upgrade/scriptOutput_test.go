package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpgradeSteps_LogScriptOutput(t *testing.T) {
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_FORMAT", "json")
	stdout := strings.Repeat("x", 1023) + "ä" + strings.Repeat("long line", 30000) + " stdout tail"
	stderr := strings.Repeat("\r", 1500) + "stderr\nwithout trailing newline"

	for _, script := range []string{core.ExposedCommandPreUpgrade, core.ExposedCommandPostUpgrade} {
		for _, scriptErr := range []error{nil, assert.AnError} {
			t.Run(fmt.Sprintf("%s/error=%v", script, scriptErr), func(t *testing.T) {
				logFile, err := os.CreateTemp(t.TempDir(), "operator-log")
				require.NoError(t, err)
				t.Cleanup(func() { assert.NoError(t, logFile.Close()) })

				// Capture the real production logger, including any sampling and encoder configuration.
				previousStderr := os.Stderr
				os.Stderr = logFile
				zapLogger := logging.NewZapLogger()
				os.Stderr = previousStderr
				ctx := logging.IntoContext(t.Context(), zapr.NewLogger(zapLogger))
				doguResource := &v2.Dogu{
					ObjectMeta: metav1.ObjectMeta{Name: "logproof", Namespace: "test"},
					Spec:       v2.DoguSpec{Version: "2.0.0"},
				}
				doguDescriptor := &core.Dogu{
					Name:            "official/logproof",
					Version:         "2.0.0",
					ExposedCommands: []core.ExposedCommand{{Name: script, Command: "/scripts/upgrade.sh"}},
				}
				podVersion := "2.0.0"
				scriptPath := "/scripts/upgrade.sh"
				if script == core.ExposedCommandPreUpgrade {
					podVersion = "1.0.0"
					scriptPath = filepath.Join(preUpgradeScriptDir, "upgrade.sh")
				}
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Name: "logproof-pod", Namespace: "test",
					Labels: map[string]string{v2.DoguLabelName: "logproof", v2.DoguLabelVersion: podVersion},
				}}
				k8sClient := fake.NewClientBuilder().WithObjects(pod).Build()
				deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
					Name: "logproof", Annotations: map[string]string{previousVersionAnnotationKey: "1.0.0"},
				}}
				deployments := newMockDeploymentInterface(t)
				deployments.EXPECT().Get(ctx, "logproof", metav1.GetOptions{}).Return(deployment, nil)
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
				executor := newMockCommandExecutor(t)
				executor.EXPECT().ExecCommandForPod(ctx, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *corev1.Pod, command exec.ShellCommand) (*bytes.Buffer, error) {
					if command.CommandWithArgs()[0] != scriptPath {
						// Binary transfers and setup commands must not opt into script logging.
						assert.Nil(t, command.Stdout())
						assert.Nil(t, command.Stderr())
						return &bytes.Buffer{}, nil
					}
					assert.Equal(t, []string{scriptPath, "1.0.0", "2.0.0"}, command.CommandWithArgs())
					if command.Stdout() != nil {
						_, err := io.WriteString(command.Stdout(), stdout[:1024])
						require.NoError(t, err)
						_, err = io.WriteString(command.Stdout(), stdout[1024:])
						require.NoError(t, err)
					}
					if command.Stderr() != nil {
						_, err := io.WriteString(command.Stderr(), stderr+"\xe2")
						require.NoError(t, err)
					}
					return bytes.NewBufferString(stdout), scriptErr
				})

				var step steps.Step
				if script == core.ExposedCommandPreUpgrade {
					factory := newMockExecPodFactory(t)
					factory.EXPECT().Exists(ctx, doguResource, doguDescriptor).Return(true)
					factory.EXPECT().CheckReady(ctx, doguResource, doguDescriptor).Return(nil)
					factory.EXPECT().Exec(ctx, doguResource, doguDescriptor, mock.Anything).Return(bytes.NewBufferString("binary archive"), nil)
					fetcher.EXPECT().FetchInstalled(ctx, cescommons.SimpleName("logproof")).Return(&core.Dogu{Version: "1.0.0"}, nil)
					upserter := NewMockResourceUpserter(t)
					if scriptErr == nil {
						upserter.EXPECT().UpsertDoguDeployment(ctx, doguResource, doguDescriptor, mock.Anything).Return(deployment, nil)
					}
					step = &UpdateDeploymentVersionStep{
						client: k8sClient, deploymentInterface: deployments, localDoguFetcher: fetcher,
						execPodFactory: factory, upserter: upserter, doguCommandExecutor: executor,
					}
				} else {
					step = &PostUpgradeStep{client: k8sClient, deploymentInterface: deployments, localDoguFetcher: fetcher, doguCommandExecutor: executor}
				}

				result := step.Run(ctx, doguResource)

				if scriptErr != nil {
					require.ErrorIs(t, result.Err, scriptErr)
				} else {
					require.NoError(t, result.Err)
					assert.Positive(t, result.RequeueAfter)
				}
				require.NoError(t, zapLogger.Sync())
				logBytes, err := os.ReadFile(logFile.Name())
				require.NoError(t, err)
				outputs := map[string]string{}
				for _, line := range bytes.Split(bytes.TrimSpace(logBytes), []byte("\n")) {
					var entry map[string]any
					require.NoError(t, json.Unmarshal(line, &entry))
					if entry["msg"] != "upgrade script output" {
						continue
					}
					assert.Less(t, len(line), 16*1024, "escaped records must leave room for container log framing")
					assert.Equal(t, "info", entry["level"])
					assert.Equal(t, "logproof", entry["dogu"])
					assert.Equal(t, "test", entry["namespace"])
					assert.Equal(t, "logproof-pod", entry["pod"])
					assert.Equal(t, script, entry["script"])
					stream := entry["stream"].(string)
					output := entry["output"].(string)
					assert.True(t, utf8.ValidString(output))
					outputs[stream] += output
				}
				require.Len(t, outputs["stdout"], len(stdout))
				assert.Equal(t, stdout, outputs["stdout"])
				// A final incomplete UTF-8 character must be flushed, using the JSON encoder's replacement character.
				assert.Equal(t, stderr+"\ufffd", outputs["stderr"])
			})
		}
	}
}
