package logging

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestFacade(t *testing.T) {
	t.Run("logs info and error", func(t *testing.T) {
		ctx, observedLogs := contextWithObservedLogger(zapcore.InfoLevel)

		Info(ctx, "installing dogu", "dogu", "nexus")
		Error(ctx, errors.New("not found"), "failed to install dogu")

		entries := observedLogs.All()
		require.Len(t, entries, 2)
		assert.Equal(t, zapcore.InfoLevel, entries[0].Level)
		assert.Equal(t, "installing dogu", entries[0].Message)
		assert.Equal(t, map[string]any{"dogu": "nexus"}, entries[0].ContextMap())
		assert.Equal(t, zapcore.ErrorLevel, entries[1].Level)
		assert.Equal(t, map[string]any{"error": "not found"}, entries[1].ContextMap())
	})

	t.Run("logs debug on debug level", func(t *testing.T) {
		ctx, observedLogs := contextWithObservedLogger(zapcore.DebugLevel)

		Debug(ctx, "requeueing dogu", "dogu", "nexus")

		entries := observedLogs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, zapcore.DebugLevel, entries[0].Level)
		assert.Equal(t, "requeueing dogu", entries[0].Message)
	})

	t.Run("discards debug on info level", func(t *testing.T) {
		ctx, observedLogs := contextWithObservedLogger(zapcore.InfoLevel)

		Debug(ctx, "requeueing dogu")

		assert.Empty(t, observedLogs.All())
	})

	t.Run("keeps the values of the context logger", func(t *testing.T) {
		ctx, observedLogs := contextWithObservedLogger(zapcore.InfoLevel)

		ctx = IntoContext(ctx, FromContext(ctx).WithName("installStep").WithValues("dogu", "nexus"))
		Info(ctx, "step finished")

		entries := observedLogs.All()
		require.Len(t, entries, 1)
		assert.Equal(t, "installStep", entries[0].LoggerName)
		assert.Equal(t, map[string]any{"dogu": "nexus"}, entries[0].ContextMap())
	})
}

func contextWithObservedLogger(level zapcore.Level) (context.Context, *observer.ObservedLogs) {
	observedCore, observedLogs := observer.New(level)
	return IntoContext(context.Background(), zapr.NewLogger(zap.New(observedCore))), observedLogs
}
