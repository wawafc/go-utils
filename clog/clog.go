package clog

import (
	"context"
	"github.com/rs/zerolog"
	"os"
	"strings"
	"time"
)

const (
	jsonF      = "json"
	prettyF    = "pretty"
	CLoggerKey = "clogger2"
)

var (
	std *zerolog.Logger
)

func New(format string) {
	switch strings.ToLower(format) {
	case jsonF:
		lg := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
		std = &lg

	default: // pretty format
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

		//output.FormatLevel = func(i interface{}) string {
		//	return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		//}
		//output.FormatMessage = func(i interface{}) string {
		//	return fmt.Sprintf("***%s****", i)
		//}
		//output.FormatFieldName = func(i interface{}) string {
		//	return fmt.Sprintf("%s:", i)
		//}
		//output.FormatFieldValue = func(i interface{}) string {
		//	return strings.ToUpper(fmt.Sprintf("%s", i))
		//}

		//output := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		//	fmt.Printf("Logger Dropped %d messages", missed)
		//})

		lg := zerolog.New(output).With().Timestamp().Caller().Logger()
		std = &lg
	}
}

func GetLog() *zerolog.Logger {
	newLog := new(zerolog.Logger)
	*newLog = *std
	return newLog
}

func GetContextLog(ctx context.Context) *zerolog.Logger {
	newLog := new(zerolog.Logger)
	ctxLog := ctx.Value(CLoggerKey).(*zerolog.Logger)
	*newLog = *ctxLog
	return newLog
}

func WithField(field map[string]interface{}) *zerolog.Logger {
	newLog := new(zerolog.Logger)
	lg := std.With().Fields(field).Logger()
	*newLog = lg
	return newLog
}
