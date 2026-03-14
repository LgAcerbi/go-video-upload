package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger is the interface used by services for structured logging.
// Controllers and repositories should use this instead of the standard log package.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	With(args ...any) Logger
	// Fatal logs at error level and exits the process with code 1.
	// Use only in main for startup failures.
	Fatal(msg string, args ...any)
}

// Config configures the logger.
type Config struct {
	// Level is the minimum level to log (default: Info).
	Level slog.Level
	// Service name to include in all log lines (e.g. "upload", "metadata").
	Service string
	// Output for log lines (default: os.Stderr).
	Output io.Writer
}

// New returns a Logger that writes structured logs (JSON) to Config.Output.
// If Config is nil, defaults are used: Level=Info, Output=os.Stderr.
func New(cfg *Config) Logger {
	if cfg == nil {
		cfg = &Config{}
	}
	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}
	h := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: cfg.Level})
	base := slog.New(h)
	if cfg.Service != "" {
		base = base.With("service", cfg.Service)
	}
	return &slogLogger{inner: base}
}

// slogLogger implements Logger using log/slog.
type slogLogger struct {
	inner *slog.Logger
}

func (l *slogLogger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *slogLogger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }
func (l *slogLogger) Debug(msg string, args ...any)  { l.inner.Debug(msg, args...) }

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{inner: l.inner.With(args...)}
}

func (l *slogLogger) Fatal(msg string, args ...any) {
	l.inner.Error(msg, args...)
	os.Exit(1)
}
