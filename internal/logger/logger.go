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
	var handlers []slog.Handler

	handlers = append(handlers, tint.NewTextHandler(os.Stderr, nil))
	handlers = append(handlers, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	Log = Logger{
		logger: slog.New(
			slogmulti.Fanout(handlers...),
		),
	}

	Log.Info("Logger Initialized")
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
