package observability

import (
	"bytes"
	"context"
	"testing"
)

func BenchmarkLogger_Info(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("Test message")
	}
}

func BenchmarkLogger_InfoWithArgs(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("Test message {Count} {Status}", i, "ok")
	}
}

func BenchmarkLogger_InfoContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "Test message")
	}
}

func BenchmarkLogger_Debug_Filtered(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel) // Debug will be filtered

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Debug("Filtered debug message")
	}
}

func BenchmarkLogger_ForContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		childLogger := logger.ForContext("request_id", "12345")
		childLogger.Info("Test message")
	}
}

func BenchmarkLogger_MultipleProperties(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
			logger.Verbose("Verbose message")
		}
	})

	b.Run("Debug", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Debug("Debug message")
		}
	})

	b.Run("Info", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info("Info message")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Warn("Warning message")
		}
	})

	b.Run("Error", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Error("Error message")
		}
	})
}

func BenchmarkNullLogger(b *testing.B) {
	logger := NewNullLogger()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("This should have zero overhead")
	}
}
