package api

import (
	"log/slog"
)

// Logger defines the logging interface.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// slogLogger wraps slog.Logger as a Logger.
type slogLogger struct {
	*slog.Logger
}

// NewSlogLogger creates a Logger from slog.Logger.
func NewSlogLogger(l *slog.Logger) Logger {
	return &slogLogger{Logger: l}
}

// nopLogger is a no-op logger.
type nopLogger struct{}

func (n *nopLogger) Debug(msg string, args ...any) {}
func (n *nopLogger) Info(msg string, args ...any)  {}
func (n *nopLogger) Warn(msg string, args ...any)  {}
func (n *nopLogger) Error(msg string, args ...any) {}

// DefaultLogger returns a default slog-based logger.
func DefaultLogger() Logger {
	return &slogWrapper{
		logger: slog.Default(),
	}
}

// slogWrapper wraps slog.Logger.
type slogWrapper struct {
	logger *slog.Logger
}

func (s *slogWrapper) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *slogWrapper) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *slogWrapper) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *slogWrapper) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}
