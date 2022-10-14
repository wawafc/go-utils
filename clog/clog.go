package clog

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"os"
	"strings"
	"time"
)

const (
	FormatPretty      = ""
	FormatJson        = "json"
	FormatJsonAndFile = "json_file"
	CLoggerKey        = "clogger2"
)

var (
	std *zerolog.Logger
)

func New(format string, debug bool) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	switch strings.ToLower(format) {
	case FormatJson:
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		lg := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
		std = &lg

	case FormatJsonAndFile:
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		fileLog := ConfigFile{
			EnableTimeKey: true,
			TimeKey:       "200601021504",
			Path:          "./logs",
			MaxSize:       "1kb",
			MaxBackups:    0,
			MaxAge:        5,     //days
			Compress:      false, // disabled by default
		}

		multi := zerolog.MultiLevelWriter(os.Stdout, NewLogFile(fileLog))

		lg := zerolog.New(multi).With().Timestamp().Caller().Logger()
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

		multi := zerolog.MultiLevelWriter(output)

		lg := zerolog.New(multi).With().Timestamp().Caller().Logger()
		std = &lg
	}
}

func GetLog() *zerolog.Logger {
	newLog := new(zerolog.Logger)
	lg := std.With().Logger()
	*newLog = lg
	return newLog
}

func GetContextLog(ctx context.Context) *zerolog.Logger {
	ctxLog := ctx.Value(CLoggerKey).(*zerolog.Logger)
	return ctxLog
}

func WithField(field map[string]interface{}) *zerolog.Logger {
	newLog := new(zerolog.Logger)
	lg := std.With().Fields(field).Logger()
	*newLog = lg
	return newLog
}

func TraceLoggingMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		return nil
	}
}

func generator() string {
	//timeNow := time.Now().Unix()
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("ffffffff")
		//return fmt.Sprintf("%v", timeNow)
	}
	return fmt.Sprintf("%x", b)
}
