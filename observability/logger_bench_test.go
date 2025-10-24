package observability

import (
	"bytes"
	"context"
	"testing"
)

func BenchmarkLogger_Info(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ReportAllocs()

	for b.Loop() {
		logger.Info("Test message")
	}
}

func BenchmarkLogger_InfoWithArgs(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ReportAllocs()

	for b.Loop() {
		logger.Info("Test message {Count} {Status}", 42, "ok")
	}
}

func BenchmarkLogger_InfoContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)
	ctx := context.Background()

	b.ReportAllocs()

	for b.Loop() {
		logger.InfoContext(ctx, "Test message")
	}
}

func BenchmarkLogger_Debug_Filtered(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel) // Debug will be filtered

	b.ReportAllocs()

	for b.Loop() {
		logger.Debug("Filtered debug message")
	}
}

func BenchmarkLogger_ForContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ReportAllocs()

	for b.Loop() {
		childLogger := logger.ForContext("request_id", "12345")
		childLogger.Info("Test message")
	}
}

func BenchmarkLogger_MultipleProperties(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ReportAllocs()

	for b.Loop() {
		logger.
			WithProperty("user_id", 12345).
			WithProperty("session_id", "abc-123").
			WithProperty("ip", "192.168.1.1").
			Info("User action {Action}", "login")
	}
}

func BenchmarkLogger_AllLevels(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, VerboseLevel)

	b.Run("Verbose", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			logger.Verbose("Verbose message")
		}
	})

	b.Run("Debug", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			logger.Debug("Debug message")
		}
	})

	b.Run("Info", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			logger.Info("Info message")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			logger.Warn("Warning message")
		}
	})

	b.Run("Error", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			logger.Error("Error message")
		}
	})
}

func BenchmarkNullLogger(b *testing.B) {
	logger := NewNullLogger()

	b.ReportAllocs()

	for b.Loop() {
		logger.Info("This should have zero overhead")
	}
}
