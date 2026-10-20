package helmfile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestHelmfileCommand(t *testing.T) {
	for _, tc := range []struct {
		name, script    string
		fail, truncated bool
	}{
		{name: "success", script: "exit 0"},
		{name: "short diagnostic", script: "printf 'diagnostic' >&2; exit 7", fail: true},
		{name: "bounded diagnostic", script: "i=0; while [ \"$i\" -lt 5000 ]; do printf 'noise'; i=$((i + 1)); done; printf 'diagnostic' >&2; exit 7", fail: true, truncated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, "helmfile")
			require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\n"+tc.script+"\n"), 0700))
			var logs strings.Builder
			ctx := log.IntoContext(context.Background(), funcr.New(func(_, message string) { logs.WriteString(message) }, funcr.Options{}))
			options := Options{HelmfileGlobalConfig: config.HelmfileGlobalConfig{HelmfileBin: binary, HelmfileUnpackDir: dir}}
			err := options.Run(ctx, "apply")
			if !tc.fail {
				require.NoError(t, err)
				assert.Contains(t, logs.String(), "completed")
				return
			}
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, 7, exitErr.ExitCode())
			assert.NotContains(t, err.Error(), "diagnostic")
			assert.Contains(t, logs.String(), "diagnostic")
			assert.Less(t, logs.Len(), 8192)
			if tc.truncated {
				assert.Contains(t, logs.String(), `"outputTruncated"=true`)
			} else {
				assert.Contains(t, logs.String(), `"outputTruncated"=false`)
			}
		})
	}
}
