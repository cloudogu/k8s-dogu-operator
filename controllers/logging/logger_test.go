package logging

import (
	"os"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/controllers/logging/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestConfigureLogger(t *testing.T) {
	originalControllerLogger := ctrl.Log
	originalLibraryLogger := core.GetLogger()
	defer func() {
		ctrl.Log = originalControllerLogger
		core.GetLogger = func() core.Logger {
			return originalLibraryLogger
		}
	}()

	t.Run("create logger with no log level set in env -> should use default", func(t *testing.T) {
		// given
		_ = os.Unsetenv(logLevelEnvVar)

		// when
		err := ConfigureLogger()

		// then
		assert.NoError(t, err)
	})

	t.Run("create logger with log level INFO", func(t *testing.T) {
		// given
		_ = os.Setenv(logLevelEnvVar, "INFO")

		// when
		err := ConfigureLogger()

		// then
		core.GetLogger().Info("test")
		assert.NoError(t, err)
	})

	t.Run("create logger with invalid log level TEST_LEVEL", func(t *testing.T) {
		// given
		_ = os.Setenv(logLevelEnvVar, "TEST_LEVEL")

		// when
		err := ConfigureLogger()

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value of log environment variable [LOG_LEVEL] is not a valid log level")
	})
}

func Test_libraryLogger_Debug(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", debugLevel, "[testLogger] test debug call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	logger.Debug("test debug call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Debugf(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", debugLevel, "[testLogger] myText - test debug call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	text := "myText"
	logger.Debugf("%s - %s", text, "test debug call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Error(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", errorLevel, "[testLogger] test error call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	logger.Error("test error call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Errorf(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", errorLevel, "[testLogger] myText - test error call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	text := "myText"
	logger.Errorf("%s - %s", text, "test error call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Info(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", infoLevel, "[testLogger] test info call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	logger.Info("test info call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Infof(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", infoLevel, "[testLogger] myText - test info call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	text := "myText"
	logger.Infof("%s - %s", text, "test info call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Warning(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", warningLevel, "[testLogger] test warning call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	logger.Warning("test warning call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}

func Test_libraryLogger_Warningf(t *testing.T) {
	// given
	loggerSink := &mocks.LogSink{}
	loggerSink.On("Info", warningLevel, "[testLogger] myText - test warning call")
	logger := libraryLogger{name: "testLogger", logger: loggerSink}

	// when
	text := "myText"
	logger.Warningf("%s - %s", text, "test warning call")

	// then
	mock.AssertExpectationsForObjects(t, loggerSink)
}
