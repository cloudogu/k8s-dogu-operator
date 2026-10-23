package logging

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRedirectLibraryLogging(t *testing.T) {
	t.Run("routes the libraries into the logger", func(t *testing.T) {
		observedLogs, zapLogger := observedLogger(t, zapcore.DebugLevel)

		redirectLibraryLogging(zapLogger)

		core.GetLogger().Error("version is broken")
		core.GetLogger().Debugf("comparing %s", "versions")
		apply.GetLogger().Debug("Applying K8s resource")

		entries := observedLogs.All()
		require.Len(t, entries, 3)

		assert.Equal(t, zapcore.ErrorLevel, entries[0].Level)
		assert.Equal(t, "version is broken", entries[0].Message)
		assert.Equal(t, map[string]any{"library": "cesapp-lib"}, entries[0].ContextMap())

		assert.Equal(t, zapcore.DebugLevel, entries[1].Level)
		assert.Equal(t, "comparing versions", entries[1].Message)

		assert.Equal(t, zapcore.DebugLevel, entries[2].Level)
		assert.Equal(t, map[string]any{"library": "k8s-apply-lib"}, entries[2].ContextMap())
	})

	t.Run("logs library warnings as warnings", func(t *testing.T) {
		observedLogs, zapLogger := observedLogger(t, zapcore.DebugLevel)

		redirectLibraryLogging(zapLogger)
		core.GetLogger().Warning("take care")
		core.GetLogger().Warningf("take %s", "care")

		entries := observedLogs.All()
		require.Len(t, entries, 2)
		assert.Equal(t, zapcore.WarnLevel, entries[0].Level)
		assert.Equal(t, zapcore.WarnLevel, entries[1].Level)
		assert.Equal(t, "take care", entries[1].Message)
	})

	t.Run("does not log library debug messages on info level", func(t *testing.T) {
		observedLogs, zapLogger := observedLogger(t, zapcore.InfoLevel)

		redirectLibraryLogging(zapLogger)
		apply.GetLogger().Debug("Applying K8s resource")

		assert.Empty(t, observedLogs.All())
	})
}

// observedLogger creates a logger whose entries can be inspected and restores the logger
// functions of the libraries afterwards so that the tests stay independent of each other.
func observedLogger(t *testing.T, level zapcore.Level) (*observer.ObservedLogs, *zap.Logger) {
	originalCoreLogger, originalApplyLogger := core.GetLogger, apply.GetLogger
	t.Cleanup(func() {
		core.GetLogger, apply.GetLogger = originalCoreLogger, originalApplyLogger
	})

	observedCore, observedLogs := observer.New(level)
	return observedLogs, zap.New(observedCore)
}
