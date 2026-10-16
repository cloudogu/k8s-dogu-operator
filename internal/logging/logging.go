package logging

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// debugVerbosity is the logr verbosity of debug log calls. It is enabled by the log levels debug and trace.
const debugVerbosity = 1

// FromContext returns the logger that is stored in the given context.
// Use it when a logger has to be enriched with WithName or WithValues, otherwise prefer Info, Debug and Error.
func FromContext(ctx context.Context) logr.Logger {
	return log.FromContext(ctx)
}

// Log returns the root logger of the operator. Use it only in code that has no context at hand,
// everywhere else the context logger carries the values of the current reconciliation.
func Log() logr.Logger {
	return log.Log
}

// IntoContext returns a new context that contains the given logger.
func IntoContext(ctx context.Context, logger logr.Logger) context.Context {
	return log.IntoContext(ctx, logger)
}

// Info logs a message that may be of general interest but does not state any danger.
func Info(ctx context.Context, message string, keysAndValues ...any) {
	FromContext(ctx).Info(message, keysAndValues...)
}

// Debug logs a message that generally should not be visible within a production environment but is
// useful when trying to pinpoint error situations.
func Debug(ctx context.Context, message string, keysAndValues ...any) {
	FromContext(ctx).V(debugVerbosity).Info(message, keysAndValues...)
}

// DebugEnabled returns true if debug logging is enabled.
// Use it to avoid building expensive log arguments that would be discarded anyway.
func DebugEnabled(ctx context.Context) bool {
	return FromContext(ctx).V(debugVerbosity).Enabled()
}

// Error logs an error that may jeopardize the success during the run-time.
func Error(ctx context.Context, err error, message string, keysAndValues ...any) {
	FromContext(ctx).Error(err, message, keysAndValues...)
}
