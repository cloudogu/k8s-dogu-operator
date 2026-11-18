package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewZapLogger_levels(t *testing.T) {
	tests := []struct {
		logLevel      string
		expectedLevel zapcore.Level
		// verbosities that have to be enabled on the resulting logr logger
		enabledVerbosities []int
		disabledVerbosity  int
	}{
		{logLevel: "", expectedLevel: zapcore.InfoLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
		{logLevel: "  ", expectedLevel: zapcore.InfoLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
		{logLevel: "info", expectedLevel: zapcore.InfoLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
		{logLevel: "INFO", expectedLevel: zapcore.InfoLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
		{logLevel: "debug", expectedLevel: zapcore.DebugLevel, enabledVerbosities: []int{0, 1}, disabledVerbosity: 2},
		// trace is kept as an alias of debug for older configurations
		{logLevel: "trace", expectedLevel: zapcore.DebugLevel, enabledVerbosities: []int{0, 1}, disabledVerbosity: 2},
		{logLevel: "warn", expectedLevel: zapcore.WarnLevel, enabledVerbosities: nil, disabledVerbosity: 0},
		{logLevel: "error", expectedLevel: zapcore.ErrorLevel, enabledVerbosities: nil, disabledVerbosity: 0},
		// invalid values fall back to the default level instead of failing the operator
		{logLevel: "garbage", expectedLevel: defaultLogLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
		{logLevel: "3", expectedLevel: defaultLogLevel, enabledVerbosities: []int{0}, disabledVerbosity: 1},
	}

	for _, test := range tests {
		t.Run(test.logLevel, func(t *testing.T) {
			t.Setenv(logLevelEnvVar, test.logLevel)

			zapLogger := NewZapLogger()

			assert.True(t, zapLogger.Core().Enabled(test.expectedLevel))
			logger := zapr.NewLogger(zapLogger)
			for _, verbosity := range test.enabledVerbosities {
				assert.True(t, logger.V(verbosity).Enabled(), "V(%d) should be enabled for log level %q", verbosity, test.logLevel)
			}
			assert.False(t, logger.V(test.disabledVerbosity).Enabled(), "V(%d) should be disabled for log level %q", test.disabledVerbosity, test.logLevel)
		})
	}
}

// fixedLogClock keeps every entry in the same sampling interval without timing-dependent assertions.
type fixedLogClock struct{ zapcore.Clock }

func (fixedLogClock) Now() time.Time { return time.Unix(1, 0) }

func TestNewZapLogger_sampling(t *testing.T) {
	t.Setenv(logLevelEnvVar, "info")
	t.Setenv(logFormatEnvVar, "json")
	logFile, err := os.CreateTemp(t.TempDir(), "operator-log")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, logFile.Close()) })

	previousStderr := os.Stderr
	os.Stderr = logFile
	zapLogger := NewZapLogger().WithOptions(zap.WithClock(fixedLogClock{zapcore.DefaultClock}))
	os.Stderr = previousStderr
	logger := zapr.NewLogger(zapLogger).WithName("upgrade").WithValues("dogu", "testdogu")
	logger = logger.WithValues("stream", "stdout")

	for i := range 250 {
		logger.Info("repeated operator message", "iteration", i)
		logger.Info("upgrade script output", "output", i)
	}

	require.NoError(t, zapLogger.Sync())
	logBytes, err := os.ReadFile(logFile.Name())
	require.NoError(t, err)
	counts := map[string]int{}
	var outputs []int
	for _, line := range bytes.Split(bytes.TrimSpace(logBytes), []byte("\n")) {
		var entry struct {
			Message string `json:"msg"`
			Logger  string `json:"logger"`
			Dogu    string `json:"dogu"`
			Stream  string `json:"stream"`
			Output  int    `json:"output"`
		}
		require.NoError(t, json.Unmarshal(line, &entry))
		assert.Equal(t, "upgrade", entry.Logger)
		assert.Equal(t, "testdogu", entry.Dogu)
		assert.Equal(t, "stdout", entry.Stream)
		counts[entry.Message]++
		if entry.Message == "upgrade script output" {
			outputs = append(outputs, entry.Output)
		}
	}
	assert.Equal(t, 101, counts["repeated operator message"])
	require.Len(t, outputs, 250)
	for i, output := range outputs {
		assert.Equal(t, i, output)
	}
}

func TestNewZapLogger_formats(t *testing.T) {
	t.Run("creates a json encoder by default", func(t *testing.T) {
		t.Setenv(logFormatEnvVar, "")

		encoder, err := encoderFromEnv()

		require.NoError(t, err)
		buffer, err := encoder.EncodeEntry(zapcore.Entry{Message: "hello"}, nil)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), `"msg":"hello"`)
	})

	t.Run("creates a json encoder", func(t *testing.T) {
		t.Setenv(logFormatEnvVar, "json")

		encoder, err := encoderFromEnv()

		require.NoError(t, err)
		buffer, err := encoder.EncodeEntry(zapcore.Entry{Message: "hello"}, nil)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), `"msg":"hello"`)
	})

	t.Run("creates a console encoder", func(t *testing.T) {
		t.Setenv(logFormatEnvVar, "text")

		encoder, err := encoderFromEnv()

		require.NoError(t, err)
		buffer, err := encoder.EncodeEntry(zapcore.Entry{Message: "hello"}, nil)
		require.NoError(t, err)
		assert.NotContains(t, buffer.String(), `"msg":`)
		assert.Contains(t, buffer.String(), "hello")
	})

	t.Run("falls back to the default format on an invalid value", func(t *testing.T) {
		t.Setenv(logFormatEnvVar, "yaml")

		encoder, err := encoderFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, "is not a valid log format: [yaml]")
		buffer, encodeErr := encoder.EncodeEntry(zapcore.Entry{Message: "hello"}, nil)
		require.NoError(t, encodeErr)
		assert.Contains(t, buffer.String(), `"msg":"hello"`)
	})
}
