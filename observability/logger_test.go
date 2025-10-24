package observability

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestLogger_BasicLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, DebugLevel)

	log.Info("Test message")

	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Output missing message: %s", output)
	}
}

func TestLogger_StructuredProperties(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	log.Info("Package {PackageId} version {Version}", "Newtonsoft.Json", "13.0.3")

	output := buf.String()
	if !strings.Contains(output, "Newtonsoft.Json") {
		t.Errorf("Output missing PackageId: %s", output)
	}
	if !strings.Contains(output, "13.0.3") {
		t.Errorf("Output missing Version: %s", output)
	}
}

func TestLogger_ForContext(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	scopedLog := log.ForContext("Source", "nuget.org")
	scopedLog.Info("Message from scoped logger with {Value}", 42)

	output := buf.String()
	// The console sink may not render all properties in default template
	// But it should at least render the message template properties
	if !strings.Contains(output, "42") {
		t.Errorf("Output missing template property: %s", output)
	}
	// Note: ForContext properties may not appear in console output without a custom template
}

func TestLogger_ContextAware(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	log.InfoContext(ctx, "Context-aware message")

	output := buf.String()
	if !strings.Contains(output, "Context-aware message") {
		t.Errorf("Output missing message: %s", output)
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name          string
		level         LogLevel
		logFunc       func(Logger)
		shouldContain bool
	}{
		{
			name:  "Info level allows Info",
			level: InfoLevel,
			logFunc: func(l Logger) {
				l.Info("Info message")
			},
			shouldContain: true,
		},
		{
			name:  "Info level blocks Debug",
			level: InfoLevel,
			logFunc: func(l Logger) {
				l.Debug("Debug message")
			},
			shouldContain: false,
		},
		{
			name:  "Debug level allows Debug",
			level: DebugLevel,
			logFunc: func(l Logger) {
				l.Debug("Debug message")
			},
			shouldContain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log := NewLogger(buf, tt.level)

			tt.logFunc(log)

			output := buf.String()
			contains := len(output) > 0

			if contains != tt.shouldContain {
				t.Errorf("Message presence = %v, want %v. Output: %s", contains, tt.shouldContain, output)
			}
		})
	}
}

func TestLogger_AllLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, VerboseLevel)

	// Test all log levels
	log.Verbose("Verbose message")
	log.Debug("Debug message")
	log.Info("Info message")
	log.Warn("Warn message")
	log.Error("Error message")

	output := buf.String()
	if !strings.Contains(output, "Verbose message") {
		t.Errorf("Output missing verbose message")
	}
	if !strings.Contains(output, "Debug message") {
		t.Errorf("Output missing debug message")
	}
	if !strings.Contains(output, "Info message") {
		t.Errorf("Output missing info message")
	}
	if !strings.Contains(output, "Warn message") {
		t.Errorf("Output missing warn message")
	}
	if !strings.Contains(output, "Error message") {
		t.Errorf("Output missing error message")
	}
}

func TestLogger_AllContextLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, VerboseLevel)
	ctx := context.Background()

	// Test all context-aware log levels
	log.VerboseContext(ctx, "Verbose context message")
	log.DebugContext(ctx, "Debug context message")
	log.InfoContext(ctx, "Info context message")
	log.WarnContext(ctx, "Warn context message")
	log.ErrorContext(ctx, "Error context message")
	log.FatalContext(ctx, "Fatal context message")

	output := buf.String()
	if !strings.Contains(output, "Verbose context message") {
		t.Errorf("Output missing verbose context message")
	}
	if !strings.Contains(output, "Debug context message") {
		t.Errorf("Output missing debug context message")
	}
	if !strings.Contains(output, "Info context message") {
		t.Errorf("Output missing info context message")
	}
	if !strings.Contains(output, "Warn context message") {
		t.Errorf("Output missing warn context message")
	}
	if !strings.Contains(output, "Error context message") {
		t.Errorf("Output missing error context message")
	}
	if !strings.Contains(output, "Fatal context message") {
		t.Errorf("Output missing fatal context message")
	}
}

func TestLogger_FatalLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, VerboseLevel)

	log.Fatal("Fatal error occurred")

	output := buf.String()
	if !strings.Contains(output, "Fatal error occurred") {
		t.Errorf("Output missing fatal message: %s", output)
	}
}

func TestLogger_WithProperty(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	scopedLog := log.WithProperty("RequestId", "12345")
	scopedLog.Info("Request processed with {Status}", "success")

	output := buf.String()
	if !strings.Contains(output, "success") {
		t.Errorf("Output missing status: %s", output)
	}
}

func TestNewDefaultLogger(t *testing.T) {
	log := NewDefaultLogger()
	if log == nil {
		t.Error("NewDefaultLogger returned nil")
	}

	// Verify it doesn't panic when used
	log.Info("Test message from default logger")
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		level     LogLevel
		logFunc   func(Logger)
		shouldLog bool
	}{
		{"Verbose level logs Verbose", VerboseLevel, func(l Logger) { l.Verbose("msg") }, true},
		{"Debug level blocks Verbose", DebugLevel, func(l Logger) { l.Verbose("msg") }, false},
		{"Info level blocks Debug", InfoLevel, func(l Logger) { l.Debug("msg") }, false},
		{"Warn level blocks Info", WarnLevel, func(l Logger) { l.Info("msg") }, false},
		{"Error level blocks Warn", ErrorLevel, func(l Logger) { l.Warn("msg") }, false},
		{"Fatal level blocks Error", FatalLevel, func(l Logger) { l.Error("msg") }, false},
		{"Warn level allows Error", WarnLevel, func(l Logger) { l.Error("msg") }, true},
		{"Info level allows Warn", InfoLevel, func(l Logger) { l.Warn("msg") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log := NewLogger(buf, tt.level)

			tt.logFunc(log)

			hasOutput := len(buf.String()) > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("Expected output=%v, got output=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

func TestNullLogger(t *testing.T) {
	log := NewNullLogger()

	// Should not panic on any method
	log.Verbose("verbose")
	log.VerboseContext(context.Background(), "verbose ctx")
	log.Debug("debug")
	log.DebugContext(context.Background(), "debug ctx")
	log.Info("info")
	log.InfoContext(context.Background(), "info ctx")
	log.Warn("warn")
	log.WarnContext(context.Background(), "warn ctx")
	log.Error("error")
	log.ErrorContext(context.Background(), "error ctx")
	log.Fatal("fatal")
	log.FatalContext(context.Background(), "fatal ctx")

	// Test scoped methods
	scopedLog := log.ForContext("key", "value")
	scopedLog.Info("Scoped logger message")

	withProp := log.WithProperty("prop", "val")
	withProp.Info("With property message")

	// No assertions - just verify no panic
}
