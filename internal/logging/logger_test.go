package logging

import (
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
