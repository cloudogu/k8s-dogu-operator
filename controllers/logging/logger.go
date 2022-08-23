package logging

import (
	"fmt"
	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

const namespaceLogLevel = "LOG_LEVEL"

const (
	printLevel int = iota
	errorLevel
	warningLevel
	infoLevel
	debugLevel
)

type libraryLogger struct {
	logger logr.LogSink
	name   string
}

func (c libraryLogger) log(level int, args ...interface{}) {
	c.logger.Info(level, fmt.Sprintf("[%s] %s", c.name, fmt.Sprint(args...)))
}

func (c libraryLogger) logf(level int, format string, args ...interface{}) {
	c.logger.Info(level, fmt.Sprintf("[%s] %s", c.name, fmt.Sprintf(format, args...)))
}

func (c libraryLogger) Debug(args ...interface{}) {
	c.log(debugLevel, args...)
}

func (c libraryLogger) Info(args ...interface{}) {
	c.log(infoLevel, args...)
}

func (c libraryLogger) Warning(args ...interface{}) {
	c.log(warningLevel, args...)
}

func (c libraryLogger) Error(args ...interface{}) {
	c.log(errorLevel, args...)
}

func (c libraryLogger) Print(args ...interface{}) {
	c.log(printLevel, args...)
}

func (c libraryLogger) Debugf(format string, args ...interface{}) {
	c.logf(debugLevel, format, args...)
}

func (c libraryLogger) Infof(format string, args ...interface{}) {
	c.logf(infoLevel, format, args...)
}

func (c libraryLogger) Warningf(format string, args ...interface{}) {
	c.logf(warningLevel, format, args...)
}

func (c libraryLogger) Errorf(format string, args ...interface{}) {
	c.logf(errorLevel, format, args...)
}

func (c libraryLogger) Printf(format string, args ...interface{}) {
	c.logf(printLevel, format, args...)
}

func getLogLevelFromEnv() (logrus.Level, error) {
	logLevel, found := os.LookupEnv(namespaceLogLevel)
	if !found {
		return logrus.ErrorLevel, nil
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return logrus.ErrorLevel, fmt.Errorf("value of log environment variable [%s] is not a valid log level: %w", namespaceLogLevel, err)
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
