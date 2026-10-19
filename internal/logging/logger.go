// Package logging contains the central logger setup of the dogu operator and a small facade
// around the controller-runtime logger.
//
// Log calls should use this package instead of sigs.k8s.io/controller-runtime/pkg/log:
// use Info, Debug and Error for plain log calls and FromContext whenever a logger has to be
// enriched with WithName or WithValues.
package logging

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	crzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	logLevelEnvVar  = "LOG_LEVEL"
	logFormatEnvVar = "LOG_FORMAT"
)

const (
	formatJSON = "json"
	formatText = "text"
)

const (
	defaultLogLevel  = zapcore.InfoLevel
	defaultLogFormat = formatJSON
)

// NewZapLogger creates the zap logger that backs all logging of the operator.
// It is configured by the environment variables LOG_LEVEL and LOG_FORMAT.
func NewZapLogger() *zap.Logger {
	level, levelErr := logLevelFromEnv()
	encoder, formatErr := encoderFromEnv()

	zapLogger := crzap.NewRaw(crzap.Level(level), crzap.Encoder(encoder))

	// invalid configuration can only be reported once the logger exists
	for _, err := range []error{levelErr, formatErr} {
		if err != nil {
			zapLogger.Warn(err.Error())
		}
	}

	return zapLogger
}

// NewLogger wraps the zap logger into a logr logger for controller-runtime.
func NewLogger(zapLogger *zap.Logger) logr.Logger {
	redirectLibraryLogging(zapLogger)

	return zapr.NewLogger(zapLogger)
}

func logLevelFromEnv() (zapcore.Level, error) {
	logLevel, found := os.LookupEnv(logLevelEnvVar)
	if !found || strings.TrimSpace(logLevel) == "" {
		return defaultLogLevel, nil
	}

	return parseLogLevel(strings.TrimSpace(logLevel))
}

// parseLogLevel maps a log level to its zap counterpart.
func parseLogLevel(logLevel string) (zapcore.Level, error) {
	switch strings.ToLower(logLevel) {
	// trace is accepted for compatibility with configurations that were written when the central
	// log level debug still mapped to trace. The operator has no log calls more verbose than debug.
	case "trace", "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return defaultLogLevel, fmt.Errorf("value of log environment variable [%s] is not a valid log level: [%s], falling back to [%s]", logLevelEnvVar, logLevel, defaultLogLevel)
	}
}

func encoderFromEnv() (zapcore.Encoder, error) {
	logFormat := strings.ToLower(strings.TrimSpace(os.Getenv(logFormatEnvVar)))
	switch logFormat {
	case "":
		return newEncoder(defaultLogFormat), nil
	case formatJSON, formatText:
		return newEncoder(logFormat), nil
	default:
		return newEncoder(defaultLogFormat), fmt.Errorf("value of log environment variable [%s] is not a valid log format: [%s], falling back to [%s]", logFormatEnvVar, logFormat, defaultLogFormat)
	}
}

func newEncoder(logFormat string) zapcore.Encoder {
	if logFormat == formatText {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		return zapcore.NewConsoleEncoder(encoderConfig)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}
