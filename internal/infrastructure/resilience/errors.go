// @ds-task T1.1: Классификация retryable ошибок (DEC-004)
// Этот файл определяет интерфейс и функции для определения,
// какие ошибки можно безопасно повторять.

package resilience

import (
	"context"
	"errors"
)

// RetryableError — интерфейс для ошибок, которые можно повторять.
// Реализации ошибок могут реализовать этот интерфейс, чтобы указать,
// что повторная попытка имеет смысл.
type RetryableError interface {
	error
	// IsRetryable возвращает true, если ошибку можно повторить.
	IsRetryable() bool
}

// IsRetryable проверяет, является ли ошибка retryable.
// Возвращает false для nil ошибок и context cancellation errors.
// Возвращает true для ошибок, реализующих RetryableError с IsRetryable() == true.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation никогда не retry
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Проверяем, реализует ли ошибка RetryableError
	var retryable RetryableError
	if errors.As(err, &retryable) {
		return retryable.IsRetryable()
	}

	// По умолчанию — retryable (консервативный подход для transient errors)
	return true
}

// RetryableErrorWrapper — обёртка для ошибок с явным флагом retryable.
// Используется, когда нужно явно пометить ошибку как retryable или non-retryable.
type RetryableErrorWrapper struct {
	Err       error
	Retryable bool
}

// Error возвращает строковое представление ошибки.
func (e *RetryableErrorWrapper) Error() string {
	return e.Err.Error()
}

// Unwrap возвращает оригинальную ошибку для errors.Is/As.
func (e *RetryableErrorWrapper) Unwrap() error {
	return e.Err
}

// IsRetryable возвращает флаг retryable.
func (e *RetryableErrorWrapper) IsRetryable() bool {
	return e.Retryable
}

// WrapRetryable оборачивает ошибку с флагом retryable = true.
func WrapRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableErrorWrapper{Err: err, Retryable: true}
}

// WrapNonRetryable оборачивает ошибку с флагом retryable = false.
func WrapNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableErrorWrapper{Err: err, Retryable: false}
}
