package observability

import (
	"context"
	"io"
	"os"

	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
	"github.com/willibrandon/mtlog/sinks"
)

// Logger is the gonuget logger interface
// Wraps mtlog for structured logging with zero allocations
type Logger interface {
	// Verbose logs detailed diagnostic information
	Verbose(messageTemplate string, args ...any)
	VerboseContext(ctx context.Context, messageTemplate string, args ...any)

	// Debug logs debugging information
	Debug(messageTemplate string, args ...any)
	DebugContext(ctx context.Context, messageTemplate string, args ...any)

	// Info logs informational messages
	Info(messageTemplate string, args ...any)
	InfoContext(ctx context.Context, messageTemplate string, args ...any)

	// Warn logs warning messages
	Warn(messageTemplate string, args ...any)
	WarnContext(ctx context.Context, messageTemplate string, args ...any)

	// Error logs error messages
	Error(messageTemplate string, args ...any)
	ErrorContext(ctx context.Context, messageTemplate string, args ...any)

	// Fatal logs fatal error messages
	Fatal(messageTemplate string, args ...any)
	FatalContext(ctx context.Context, messageTemplate string, args ...any)

	// ForContext creates a child logger with additional context
	ForContext(key string, value any) Logger

	// WithProperty adds a property to the logger
	WithProperty(key string, value any) Logger
}

// mtlogAdapter wraps mtlog logger to implement gonuget Logger interface
type mtlogAdapter struct {
	logger core.Logger
}

// NewLogger creates a new gonuget logger with sensible defaults
func NewLogger(output io.Writer, level LogLevel) Logger {
	// Create console sink with writer - use WithProperties for testing
	consoleSink := sinks.NewConsoleSinkWithWriter(output)

	opts := []mtlog.Option{
		mtlog.WithSink(consoleSink),
		mtlog.WithTimestamp(),
		mtlog.WithMachineName(),
		mtlog.WithProcess(),
	}

	// Set minimum level
	switch level {
	case VerboseLevel:
		opts = append(opts, mtlog.Verbose())
	case DebugLevel:
		opts = append(opts, mtlog.Debug())
	case InfoLevel:
		opts = append(opts, mtlog.Information())
	case WarnLevel:
		opts = append(opts, mtlog.Warning())
	case ErrorLevel:
		opts = append(opts, mtlog.Error())
	case FatalLevel:
		opts = append(opts, mtlog.WithMinimumLevel(core.FatalLevel))
	}

	return &mtlogAdapter{
		logger: mtlog.New(opts...),
	}
}

// NewDefaultLogger creates a logger with console output and Info level
func NewDefaultLogger() Logger {
	return NewLogger(os.Stdout, InfoLevel)
}

// Verbose implements Logger.Verbose
func (a *mtlogAdapter) Verbose(messageTemplate string, args ...any) {
	a.logger.Verbose(messageTemplate, args...)
}

// VerboseContext implements Logger.VerboseContext
func (a *mtlogAdapter) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.VerboseContext(ctx, messageTemplate, args...)
}

// Debug implements Logger.Debug
func (a *mtlogAdapter) Debug(messageTemplate string, args ...any) {
	a.logger.Debug(messageTemplate, args...)
}

// DebugContext implements Logger.DebugContext
func (a *mtlogAdapter) DebugContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.DebugContext(ctx, messageTemplate, args...)
}

// Info implements Logger.Info
func (a *mtlogAdapter) Info(messageTemplate string, args ...any) {
	a.logger.Info(messageTemplate, args...)
}

// InfoContext implements Logger.InfoContext
func (a *mtlogAdapter) InfoContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.InfoContext(ctx, messageTemplate, args...)
}

// Warn implements Logger.Warn
func (a *mtlogAdapter) Warn(messageTemplate string, args ...any) {
	a.logger.Warn(messageTemplate, args...)
}

// WarnContext implements Logger.WarnContext
func (a *mtlogAdapter) WarnContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.WarnContext(ctx, messageTemplate, args...)
}

// Error implements Logger.Error
func (a *mtlogAdapter) Error(messageTemplate string, args ...any) {
	a.logger.Error(messageTemplate, args...)
}

// ErrorContext implements Logger.ErrorContext
func (a *mtlogAdapter) ErrorContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.ErrorContext(ctx, messageTemplate, args...)
}

// Fatal implements Logger.Fatal
func (a *mtlogAdapter) Fatal(messageTemplate string, args ...any) {
	a.logger.Fatal(messageTemplate, args...)
}

// FatalContext implements Logger.FatalContext
func (a *mtlogAdapter) FatalContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.FatalContext(ctx, messageTemplate, args...)
}

// ForContext implements Logger.ForContext
func (a *mtlogAdapter) ForContext(key string, value any) Logger {
	return &mtlogAdapter{
		logger: a.logger.ForContext(key, value),
	}
}

// WithProperty implements Logger.WithProperty (alias for ForContext)
func (a *mtlogAdapter) WithProperty(key string, value any) Logger {
	return a.ForContext(key, value)
}

// LogLevel represents log verbosity level
type LogLevel int

const (
	// VerboseLevel is the most detailed logging level.
	VerboseLevel LogLevel = iota
	// DebugLevel is for debug messages.
	DebugLevel
	// InfoLevel is for informational messages.
	InfoLevel
	// WarnLevel is for warning messages.
	WarnLevel
	// ErrorLevel is for error messages.
	ErrorLevel
	// FatalLevel is for fatal error messages.
	FatalLevel
)

// NullLogger is a logger that discards all output
type nullLogger struct{}

// NewNullLogger creates a logger that discards all output
func NewNullLogger() Logger {
	return &nullLogger{}
}

func (n *nullLogger) Verbose(messageTemplate string, args ...any)                             {}
func (n *nullLogger) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {}
func (n *nullLogger) Debug(messageTemplate string, args ...any)                               {}
func (n *nullLogger) DebugContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Info(messageTemplate string, args ...any)                                {}
func (n *nullLogger) InfoContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Warn(messageTemplate string, args ...any)                                {}
func (n *nullLogger) WarnContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Error(messageTemplate string, args ...any)                               {}
func (n *nullLogger) ErrorContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Fatal(messageTemplate string, args ...any)                               {}
func (n *nullLogger) FatalContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) ForContext(key string, value any) Logger                                 { return n }
func (n *nullLogger) WithProperty(key string, value any) Logger                               { return n }
