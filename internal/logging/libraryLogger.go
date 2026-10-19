package logging

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"go.uber.org/zap"
)

// redirectLibraryLogging routes the logging of the cesapp-lib and the k8s-apply-lib into the
// given logger.
func redirectLibraryLogging(zapLogger *zap.Logger) {
	cesappLibLogger := &libraryLogger{SugaredLogger: zapLogger.Sugar().With("library", "cesapp-lib")}
	core.GetLogger = func() core.Logger {
		return cesappLibLogger
	}

	k8sApplyLibLogger := &libraryLogger{SugaredLogger: zapLogger.Sugar().With("library", "k8s-apply-lib")}
	apply.GetLogger = func() apply.Logger {
		return k8sApplyLibLogger
	}
}

// libraryLogger implements the logger interfaces of the cesapp-lib and the k8s-apply-lib.
// The sugared logger already provides all methods except the warning ones.
type libraryLogger struct {
	*zap.SugaredLogger
}

func (ll *libraryLogger) Warning(args ...any) {
	ll.Warn(args...)
}

func (ll *libraryLogger) Warningf(format string, args ...any) {
	ll.Warnf(format, args...)
}
