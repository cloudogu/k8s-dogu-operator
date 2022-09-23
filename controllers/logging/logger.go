package logging

import (
	"fmt"
	"os"

	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

const logLevelEnvVar = "LOG_LEVEL"

const (
	errorLevel int = iota
	warningLevel
	infoLevel
	debugLevel
)

var CurrentLogLevel = logrus.ErrorLevel

type libraryLogger struct {
	logger logr.LogSink
	name   string
}

func (ll *libraryLogger) log(level int, args ...interface{}) {
	ll.logger.Info(level, fmt.Sprintf("[%s] %s", ll.name, fmt.Sprint(args...)))
}

func (ll *libraryLogger) logf(level int, format string, args ...interface{}) {
	ll.logger.Info(level, fmt.Sprintf("[%s] %s", ll.name, fmt.Sprintf(format, args...)))
}

func (ll *libraryLogger) Debug(args ...interface{}) {
	ll.log(debugLevel, args...)
}

func (ll *libraryLogger) Info(args ...interface{}) {
	ll.log(infoLevel, args...)
}

func (ll *libraryLogger) Warning(args ...interface{}) {
	ll.log(warningLevel, args...)
}

func (ll *libraryLogger) Error(args ...interface{}) {
	ll.log(errorLevel, args...)
}

func (ll *libraryLogger) Debugf(format string, args ...interface{}) {
	ll.logf(debugLevel, format, args...)
}

func (ll *libraryLogger) Infof(format string, args ...interface{}) {
	ll.logf(infoLevel, format, args...)
}

func (ll *libraryLogger) Warningf(format string, args ...interface{}) {
	ll.logf(warningLevel, format, args...)
}

func (ll *libraryLogger) Errorf(format string, args ...interface{}) {
	ll.logf(errorLevel, format, args...)
}

func getLogLevelFromEnv() (logrus.Level, error) {
	logLevel, found := os.LookupEnv(logLevelEnvVar)
	if !found {
		return logrus.ErrorLevel, nil
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return logrus.ErrorLevel, fmt.Errorf("value of log environment variable [%s] is not a valid log level: %w", logLevelEnvVar, err)
	}

	return level, nil
}

func ConfigureLogger() error {
	level, err := getLogLevelFromEnv()
	if err != nil {
		return err
	}

	// create logrus logger that can be styled and formatted
	logrusLog := logrus.New()
	logrusLog.SetFormatter(&logrus.TextFormatter{})
	logrusLog.SetLevel(level)

	CurrentLogLevel = level

	// convert logrus logger to logr logger
	logrusLogrLogger := logrusr.New(logrusLog)

	// set logr logger as controller logger
	ctrl.SetLogger(logrusLogrLogger)

	// set custom logger implementation to cesapp-lib logger
	cesappLibLogger := libraryLogger{name: "cesapp-lib", logger: logrusLogrLogger.GetSink()}
	core.GetLogger = func() core.Logger {
		return &cesappLibLogger
	}

	// set custom logger implementation to k8s-apply-lib logger
	k8sApplyLibLogger := libraryLogger{name: "k8s-apply-lib", logger: logrusLogrLogger.GetSink()}
	apply.GetLogger = func() apply.Logger {
		return &k8sApplyLibLogger
	}

	return nil
}
