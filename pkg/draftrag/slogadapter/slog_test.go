package slogadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

// @sk-test slog-otel-adapters#T2.1: all 4 LogLevel мапятся на правильный slog.Level
func TestSlogAdapter_LevelMapping(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	tests := []struct {
		level     domain.LogLevel
		wantLevel string
		msg       string
	}{
		{domain.LogLevelDebug, "DEBUG", "debug message"},
		{domain.LogLevelInfo, "INFO", "info message"},
		{domain.LogLevelWarn, "WARN", "warn message"},
		{domain.LogLevelError, "ERROR", "error message"},
	}

	for _, tt := range tests {
		buf.Reset()
		logger.Log(context.Background(), tt.level, tt.msg)

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result["level"] != tt.wantLevel {
			t.Errorf("Log(%s): got level=%v, want %s", tt.level, result["level"], tt.wantLevel)
		}
		if result["msg"] != tt.msg {
			t.Errorf("Log(%s): got msg=%v, want %s", tt.level, result["msg"], tt.msg)
		}
	}
}

// @sk-test slog-otel-adapters#T2.2: LogField.Key/Value → slog.Attr
func TestSlogAdapter_Fields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.New(slog.NewJSONHandler(&buf, nil)))

	logger.Log(context.Background(), domain.LogLevelInfo, "test",
		domain.LogField{Key: "str", Value: "val1"},
		domain.LogField{Key: "num", Value: 42},
		domain.LogField{Key: "err", Value: domain.ErrEmptyChunkID},
		domain.LogField{Key: "struct", Value: domain.LogField{Key: "nested", Value: "x"}},
	)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["str"] != "val1" {
		t.Errorf("expected str=val1, got %v", result["str"])
	}
	if n, ok := result["num"].(float64); !ok || n != 42 {
		t.Errorf("expected num=42, got %v", result["num"])
	}
}

// @sk-test slog-otel-adapters#T2.3: context со span → trace_id + span_id в JSON
func TestSlogAdapter_TraceContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.New(slog.NewJSONHandler(&buf, nil)))

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{2},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	logger.Log(ctx, domain.LogLevelInfo, "traced")

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["trace_id"] != "01000000000000000000000000000000" {
		t.Errorf("expected trace_id, got %v", result["trace_id"])
	}
	if result["span_id"] != "0200000000000000" {
		t.Errorf("expected span_id, got %v", result["span_id"])
	}
}

func TestSlogAdapter_NilContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.New(slog.NewJSONHandler(&buf, nil)))

	logger.Log(nil, domain.LogLevelInfo, "nil ctx test")

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["msg"] != "nil ctx test" {
		t.Errorf("expected msg='nil ctx test', got %v", result["msg"])
	}
}
