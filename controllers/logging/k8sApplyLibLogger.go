package logging

import (
	"fmt"
	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

const logLevelDebug = 10
const logLevelDefault = 1

type k8sApplyLibLogger struct {
	logger logr.LogSink
}

func (tl *k8sApplyLibLogger) Debug(args ...interface{}) {
	tl.logger.Info(logLevelDebug, fmt.Sprint(args...))
}

func (tl *k8sApplyLibLogger) Info(args ...interface{}) {
	tl.logger.Info(logLevelDefault, fmt.Sprint(args...))
}

func (tl *k8sApplyLibLogger) Error(args ...interface{}) {
	tl.logger.Error(fmt.Errorf(fmt.Sprint(args...)), fmt.Sprint(args...))
}

// ConfigureLogger initializes the logger used by the operator.
func ConfigureLogger() {
	// the logrus logger provides the visual representation of the logger
	logrusLog := logrus.New()
	logrusLog.SetFormatter(&logrus.TextFormatter{})
	logrusLog.SetLevel(logrus.DebugLevel)

	// the bridge logrusr transform the logrus logger into a logr logger used by the controller
	logrLogger := logrusr.New(logrusLog)

	// assign logr logger to the controller runtime
	ctrl.SetLogger(logrLogger)

	// create and assign logger used by the k8s-apply-lib
	tl := k8sApplyLibLogger{logger: logrLogger.GetSink()}
	apply.GetLogger = func() apply.Logger {
		return &tl
	}
}
