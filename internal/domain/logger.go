package domain

import "context"

// LogLevel — уровень логирования.
type LogLevel string

// Уровни логирования.
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogField — структурированное поле лог-события.
type LogField struct {
	Key   string
	Value any
}

// Logger — минимальный интерфейс структурированного логирования.
//
// Реализация должна быть thread-safe.
// Любые ошибки/паники логгера не должны ломать основной поток библиотеки:
// вызывайте логгер через SafeLog.
type Logger interface {
	Log(ctx context.Context, level LogLevel, msg string, fields ...LogField)
}

type noopLogger struct{}

func (noopLogger) Log(ctx context.Context, level LogLevel, msg string, fields ...LogField) {
	_ = ctx
	_ = level
	_ = msg
	_ = fields
}

// NoopLogger возвращает logger, который игнорирует все события.
func NoopLogger() Logger {
	return noopLogger{}
}

// SafeLog вызывает logger best-effort:
// - no-op если logger == nil
// - защищён recover, чтобы паника логгера не пробивалась наружу
func SafeLog(ctx context.Context, logger Logger, level LogLevel, msg string, fields ...LogField) {
	if logger == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	logger.Log(ctx, level, msg, fields...)
}
