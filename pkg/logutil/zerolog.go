package logutil

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"strings"
)

func InitZeroLog(ctx context.Context, level string) context.Context {
	// use unix time
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// set log level
	var logLevel zerolog.Level
	switch strings.ToLower(level) {
	case zerolog.LevelDebugValue:
		logLevel = zerolog.DebugLevel
	case zerolog.LevelInfoValue:
		logLevel = zerolog.InfoLevel
	case zerolog.LevelWarnValue:
		logLevel = zerolog.WarnLevel
	case zerolog.LevelErrorValue:
		logLevel = zerolog.ErrorLevel
	case zerolog.LevelFatalValue:
		logLevel = zerolog.FatalLevel
	default:
		logLevel = zerolog.TraceLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// show caller: github.com/rs/zerolog#add-file-and-line-number-to-log
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return fmt.Sprintf("%s:%d", short, line)
	}
	log.Logger = log.With().Caller().Logger()

	ctx = log.Logger.WithContext(ctx)
	return ctx
}
