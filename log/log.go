package log

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/Shopify/logrus-bugsnag"
	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/sirupsen/logrus"

	"github.com/Conscience/protocol/config/env"
)

type Fields = logrus.Fields

var (
	DebugLevel = logrus.DebugLevel
	InfoLevel  = logrus.InfoLevel
)

var globalFields = Fields{}
var globalFieldsMutex = &sync.RWMutex{}

func SetField(key string, value interface{}) {
	if env.BugsnagEnabled {
		globalFieldsMutex.Lock()
		defer globalFieldsMutex.Unlock()
		globalFields[key] = value
	}
}

func getFieldsCopy() Fields {
	globalFieldsMutex.RLock()
	defer globalFieldsMutex.RUnlock()

	fields := Fields{}
	for k, v := range globalFields {
		fields[k] = v
	}
	return fields
}

func init() {
	if env.BugsnagEnabled {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       env.BugsnagAPIKey,
			ReleaseStage: env.ReleaseStage,
			AppVersion:   env.AppVersion,
		})
		hook, err := logrus_bugsnag.NewBugsnagHook()
		if err != nil {
			panic(err)
		}
		logrus.AddHook(hook)
	}

	// Add some of the current environment to the log metadata
	SetField("env.PATH", os.Getenv("PATH"))
	SetField("env.PWD", os.Getenv("PWD"))
	SetField("env.CONSCIENCE_APP_PATH", os.Getenv("CONSCIENCE_APP_PATH"))
	SetField("env.CONSCIENCE_BINARIES_PATH", os.Getenv("CONSCIENCE_BINARIES_PATH"))

	if env.ConsoleLoggingEnabled == false {
		logrus.SetOutput(ioutil.Discard)
	}
}

func WithFields(fields Fields) *logrus.Entry {
	return logrus.WithFields(fields)
}

// SetLevel sets the standard logger level.
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Debug(args...)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Print(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Error(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Panic(args...)
}

// Fatal logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatal(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Fatal(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Printf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalf(format string, args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Fatalf(format, args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Errorln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Panicln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalln(args ...interface{}) {
	logrus.WithFields(getFieldsCopy()).Fatalln(args...)
}
