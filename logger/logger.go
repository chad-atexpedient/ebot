// Package logger provides structured logging utilities for ebot
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Logger is the interface for structured logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// Level represents log levels
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Config configures the logger
type Config struct {
	Level      Level
	Format     string // "json" or "text"
	Output     io.Writer
	AddSource  bool
	TimeFormat string
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:     LevelInfo,
		Format:    "json",
		Output:    os.Stdout,
		AddSource: false,
	}
}

// slogLogger wraps slog.Logger to implement our Logger interface
type slogLogger struct {
	logger *slog.Logger
	ctx    context.Context
}

var (
	defaultLogger Logger
	once          sync.Once
)

// New creates a new logger with the given configuration
func New(cfg *Config) Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(cfg.Output, opts)
	} else {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// Default returns the default logger, creating it if necessary
func Default() Logger {
	once.Do(func() {
		defaultLogger = New(DefaultConfig())
	})
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l Logger) {
	defaultLogger = l
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
		ctx:    l.ctx,
	}
}

func (l *slogLogger) WithContext(ctx context.Context) Logger {
	return &slogLogger{
		logger: l.logger,
		ctx:    ctx,
	}
}

// Context keys for logging
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	TenantIDKey  contextKey = "tenant_id"
)

// WithRequestID adds a request ID to the logger
func WithRequestID(l Logger, requestID string) Logger {
	return l.With("request_id", requestID)
}

// WithUserID adds a user ID to the logger
func WithUserID(l Logger, userID string) Logger {
	return l.With("user_id", userID)
}

// WithTenantID adds a tenant ID to the logger
func WithTenantID(l Logger, tenantID string) Logger {
	return l.With("tenant_id", tenantID)
}

// FromContext extracts logger context values and returns an enriched logger
func FromContext(ctx context.Context, l Logger) Logger {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		l = l.With("request_id", requestID)
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		l = l.With("user_id", userID)
	}
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok {
		l = l.With("tenant_id", tenantID)
	}
	return l
}

// NopLogger returns a logger that discards all output
func NopLogger() Logger {
	return &slogLogger{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}
