// Package slogadapter предоставляет адаптер log/slog для domain.Logger.
package slogadapter

import (
	"context"
	"log/slog"

	"github.com/bzdvdn/draftrag/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type adapter struct {
	logger *slog.Logger
}

// New создаёт domain.Logger из *slog.Logger.
//
// @sk-task slog-otel-adapters#T1.1: slogadapter.New — адаптер slog → domain.Logger с trace correlation
func New(logger *slog.Logger) domain.Logger {
	return &adapter{logger: logger}
}

func (a *adapter) Log(ctx context.Context, level domain.LogLevel, msg string, fields ...domain.LogField) {
	if ctx == nil {
		ctx = context.Background()
	}

	lvl := convertLevel(level)
	attrs := convertFields(fields)

	if spanCtx := trace.SpanFromContext(ctx).SpanContext(); spanCtx.HasTraceID() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	a.logger.LogAttrs(ctx, lvl, msg, attrs...)
}

func convertLevel(lvl domain.LogLevel) slog.Level {
	switch lvl {
	case domain.LogLevelDebug:
		return slog.LevelDebug
	case domain.LogLevelInfo:
		return slog.LevelInfo
	case domain.LogLevelWarn:
		return slog.LevelWarn
	case domain.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func convertFields(fields []domain.LogField) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, len(fields))
	for i, f := range fields {
		attrs[i] = slog.Any(f.Key, f.Value)
	}
	return attrs
}
