package log

import (
	"os"
	"strings"

	"github.com/Shopify/logrus-bugsnag"
	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/sirupsen/logrus"

	"github.com/Conscience/protocol/constants"
)

type Fields = logrus.Fields

var (
	DebugLevel = logrus.DebugLevel
	InfoLevel  = logrus.InfoLevel

	bugsnagEnabled = os.Getenv("BUGSNAG_ENABLED") != ""
)

var globalFields = logrus.Fields{}

func SetField(key string, value interface{}) {
	if bugsnagEnabled {
		globalFields[key] = value
	}
}

func init() {
	if bugsnagEnabled {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       constants.BugsnagAPIKey,
			ReleaseStage: constants.ReleaseStage,
			AppVersion:   constants.AppVersion,
		})
		hook, err := logrus_bugsnag.NewBugsnagHook()
		if err != nil {
			panic(err)
		}
		logrus.AddHook(hook)
	}

	// Add the current environment to the log metadata
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		SetField("env."+parts[0], parts[1])
	}
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return logrus.WithFields(globalFields).WithFields(fields)
}

// SetLevel sets the standard logger level.
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	logrus.WithFields(globalFields).Debug(args...)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	logrus.WithFields(globalFields).Print(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	logrus.WithFields(globalFields).Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	logrus.WithFields(globalFields).Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	logrus.WithFields(globalFields).Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	logrus.WithFields(globalFields).Error(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	logrus.WithFields(globalFields).Panic(args...)
}

// Fatal logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatal(args ...interface{}) {
	logrus.WithFields(globalFields).Fatal(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Printf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalf(format string, args ...interface{}) {
	logrus.WithFields(globalFields).Fatalf(format, args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	logrus.WithFields(globalFields).Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	logrus.WithFields(globalFields).Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	logrus.WithFields(globalFields).Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	logrus.WithFields(globalFields).Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	logrus.WithFields(globalFields).Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	logrus.WithFields(globalFields).Errorln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
	logrus.WithFields(globalFields).Panicln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalln(args ...interface{}) {
	logrus.WithFields(globalFields).Fatalln(args...)
}
