package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Only use the sugared logger instead of the standard, structured logger in
// order to have a more standard interface (at least for now)
var log *zap.SugaredLogger

func init() {
	// Get a config with sane defaults
	cfg := zap.NewDevelopmentConfig()
	// Don't print line numbers since we'll always be calling zap from this file
	cfg.DisableCaller = true
	logLevel := getLogLevelFromEnvironment()
	cfg.Level = zap.NewAtomicLevelAt(logLevel)

	logger, err := cfg.Build()
	if err != nil {
		// This should be a programming error
		panic(err)
	}

	zap.ReplaceGlobals(logger)

	log = zap.S()
}

// stringToLogLevel converts a log level given as a string to a zap Level.
func stringToLogLevel(s string) zapcore.Level {
	switch {
	case s == "debug":
		return zap.DebugLevel
	case s == "info":
		return zap.InfoLevel
	case s == "warn" || s == "warning":
		return zap.WarnLevel
	case s == "error":
		return zap.ErrorLevel
	case s == "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

// getLogLevelFromEnvironment gets the log level from the environment with sane
// defaulting if no value or an invalid value is specified. Note that we call
// "os" directly here because we want to be able to log in the envvars package
// without a circular dependency.
func getLogLevelFromEnvironment() zapcore.Level {
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	cloudEnv := getCloudEnvironment()
	if logLevel == "" {
		if cloudEnv == "development" {
			// Default more verbose logging for dev environment
			logLevel = "debug"
		} else {
			// Default to info for non-dev environment
			logLevel = "info"
		}
	}

	return stringToLogLevel(logLevel)
}

// getCloudEnvironment is an unfortunate artifact of not being able to use the
// envvars package here.
func getCloudEnvironment() string {
	return strings.ToLower(os.Getenv("CONTAINERSHIP_CLOUD_ENVIRONMENT"))
}

// Fatal implements the fatal logging level
func Fatal(args ...interface{}) {
	log.Fatal(args)
}

// Fatalf implements the fatal logging level with a format string
func Fatalf(s string, args ...interface{}) {
	log.Fatalf(s, args)
}

// Error implements the error logging level
func Error(args ...interface{}) {
	log.Error(args)
}

// Errorf implements the error logging level with a format string
func Errorf(s string, args ...interface{}) {
	log.Errorf(s, args)
}

// Warn implements the warn logging level
func Warn(args ...interface{}) {
	log.Warn(args)
}

// Warnf implements the warn logging level with a format string
func Warnf(s string, args ...interface{}) {
	log.Warnf(s, args)
}

// Info implements the info logging level
func Info(args ...interface{}) {
	log.Info(args)
}

// Infof implements the info logging level with a format string
func Infof(s string, args ...interface{}) {
	log.Infof(s, args)
}

// Debug implements the debug logging level
func Debug(args ...interface{}) {
	log.Debug(args)
}

// Debugf implements the debug logging level with a format string
func Debugf(s string, args ...interface{}) {
	log.Debugf(s, args)
}
