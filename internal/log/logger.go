package log

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

func NewLogger(level string) gokitlog.Logger {
	optionLevel, ok := logLevelMapping[level]

	if !ok {
		optionLevel = gokitlevel.AllowAll()
	}

	logger := gokitlog.NewJSONLogger(gokitlog.NewSyncWriter(os.Stdout))
	logger = gokitlevel.NewFilter(logger, optionLevel)
	logger = gokitlog.With(logger, "ts", gokitlog.DefaultTimestampUTC, "caller", gokitlog.DefaultCaller)
	return logger
}
