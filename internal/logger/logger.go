package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	slogmulti "github.com/samber/slog-multi"
)

var Log Logger

type Logger struct {
	logger *slog.Logger
}

func Initialize() {
	Log = New()

	Log.Info("Logger Initialized")
}

func New() Logger {
	var handlers []slog.Handler

	handlers = append(handlers, tint.NewHandler(os.Stdout, nil))
	handlers = append(handlers, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return Logger{
		logger: slog.New(
			slogmulti.Fanout(handlers...),
		),
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}
