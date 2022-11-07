package log

import (
	"os"

	"github.com/go-kit/log"
	loglevel "github.com/go-kit/log/level"
)

const (
	levelAll   string = "all"
	levelDebug string = "debug"
	levelInfo  string = "info"
	levelWarn  string = "warn"
	levelError string = "error"
)

var logLevelMapping = map[string]loglevel.Option{
	levelAll:   loglevel.AllowAll(),
	levelDebug: loglevel.AllowDebug(),
	levelInfo:  loglevel.AllowInfo(),
	levelWarn:  loglevel.AllowWarn(),
	levelError: loglevel.AllowError(),
}

func NewLogger(level string) log.Logger {
	optionLevel, ok := logLevelMapping[level]

	if !ok {
		optionLevel = loglevel.AllowAll()
	}

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = loglevel.NewFilter(logger, optionLevel)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	return logger
}
