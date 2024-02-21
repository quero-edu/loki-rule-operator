package logger

import (
	"os"

	gokitlog "github.com/go-kit/log"
	gokitlevel "github.com/go-kit/log/level"
)

const (
	levelAll   string = "all"
	levelDebug string = "debug"
	levelInfo  string = "info"
	levelWarn  string = "warn"
	levelError string = "error"
)

var logLevelMapping = map[string]gokitlevel.Option{
	levelAll:   gokitlevel.AllowAll(),
	levelDebug: gokitlevel.AllowDebug(),
	levelInfo:  gokitlevel.AllowInfo(),
	levelWarn:  gokitlevel.AllowWarn(),
	levelError: gokitlevel.AllowError(),
}

type loggerImpl struct {
	logger        gokitlog.Logger
	errorCallback func(err error, args ...interface{})
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(err error, msg string, args ...interface{})
}

func (l *loggerImpl) Debug(msg string, args ...interface{}) {
	err := gokitlevel.Debug(l.logger).Log("msg", msg, "args", args)
	if err != nil {
		l.errorCallback(err, args)
	}
}

func (l *loggerImpl) Info(msg string, args ...interface{}) {
	err := gokitlevel.Info(l.logger).Log("msg", msg, "args", args)
	if err != nil {
		l.errorCallback(err, args)
	}
}

func (l *loggerImpl) Warn(msg string, args ...interface{}) {
	err := gokitlevel.Warn(l.logger).Log("msg", msg, "args", args)
	if err != nil {
		l.errorCallback(err, args)
	}
}

func (l *loggerImpl) Error(logError error, msg string, args ...interface{}) {
	err := gokitlevel.Error(l.logger).Log("msg", msg, "err", logError.Error(), "args", args)
	if err != nil {
		l.errorCallback(err, args)
	}
}

func NewLogger(level string, onError func(err error, args ...interface{})) Logger {
	optionLevel, ok := logLevelMapping[level]

	if !ok {
		optionLevel = gokitlevel.AllowAll()
	}

	logger := gokitlog.NewJSONLogger(gokitlog.NewSyncWriter(os.Stdout))
	logger = gokitlevel.NewFilter(logger, optionLevel)
	logger = gokitlog.With(logger, "ts", gokitlog.DefaultTimestampUTC, "caller", gokitlog.DefaultCaller)

	return &loggerImpl{logger: logger, errorCallback: onError}
}

func NewNopLogger() Logger {
	noopErrorCallback := func(_ error, _ ...interface{}) {}

	return &loggerImpl{logger: gokitlog.NewNopLogger(), errorCallback: noopErrorCallback}
}
