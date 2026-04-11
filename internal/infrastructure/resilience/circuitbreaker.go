// @ds-task T2.1: Circuit breaker state machine (AC-003, AC-004, DEC-002)
// Этот файл реализует state machine circuit breaker с состояниями
// closed, open и half-open. Thread-safe через sync.RWMutex.

package resilience

import (
	"errors"
	"sync"
	"time"
)

// CircuitState определяет состояние circuit breaker.
type CircuitState int

const (
	// CircuitClosed — нормальная работа, запросы выполняются.
	CircuitClosed CircuitState = iota
	// CircuitOpen — блокировка, запросы немедленно отклоняются.
	CircuitOpen
	// CircuitHalfOpen — пробное восстановление, один запрос разрешён.
	CircuitHalfOpen
)

// String возвращает строковое представление состояния.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker реализует state machine для защиты от каскадных отказов.
// Thread-safe для использования из множества goroutines.
type CircuitBreaker struct {
	// threshold — порог ошибок для перехода в open.
	threshold int

	// timeout — время восстановления для перехода в half-open.
	timeout time.Duration

	// mu защищает состояние.
	mu sync.RWMutex

	// state — текущее состояние.
	state CircuitState

	// failureCount — счётчик ошибок в текущем окне.
	failureCount int

	// lastFailureTime — время последней ошибки.
	lastFailureTime time.Time

	// probeSent — флаг отправленного probe-запроса в состоянии half-open.
	// Защищён mu; сбрасывается при выходе из half-open.
	probeSent bool
}

// CircuitBreakerConfig содержит настройки circuit breaker.
type CircuitBreakerConfig struct {
	// Threshold — порог ошибок для перехода в open (default: 5).
	Threshold int

	// Timeout — время восстановления для перехода в half-open (default: 30s).
	Timeout time.Duration
}

// DefaultCircuitBreakerConfig возвращает конфигурацию с разумными значениями.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Threshold: 5,
		Timeout:   30 * time.Second,
	}
}

// NewCircuitBreaker создаёт новый circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		threshold: config.Threshold,
		timeout:   config.Timeout,
		state:     CircuitClosed,
	}
}

// ErrCircuitOpen возвращается, когда circuit breaker в состоянии open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CanExecute проверяет, можно ли выполнить запрос.
// Возвращает nil, если запрос разрешён, и ErrCircuitOpen если нет.
// Автоматически переходит из open в half-open по таймауту.
func (cb *CircuitBreaker) CanExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return nil

	case CircuitOpen:
		// Проверяем, пора ли переходить в half-open
		if time.Since(cb.lastFailureTime) >= cb.timeout {
			cb.state = CircuitHalfOpen
			cb.probeSent = true // первая probe уже выдана переходящей горутине
			return nil
		}
		return ErrCircuitOpen

	case CircuitHalfOpen:
		// В half-open пропускаем ровно один probe-запрос
		if cb.probeSent {
			return ErrCircuitOpen
		}
		cb.probeSent = true
		return nil

	default:
		return ErrCircuitOpen
	}
}

// RecordSuccess фиксирует успешный результат запроса.
// При half-open переходит в closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		// Успешный probe — восстанавливаемся
		cb.state = CircuitClosed
		cb.failureCount = 0
		cb.probeSent = false

	case CircuitClosed:
		// Сбрасываем счётчик ошибок при успехе
		cb.failureCount = 0
	}
}

// RecordFailure фиксирует ошибку запроса.
// Увеличивает счётчик, при превышении threshold переходит в open.
// При half-open сразу возвращается в open.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitHalfOpen:
		// Неудачный probe — снова в open
		cb.state = CircuitOpen
		cb.probeSent = false

	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.threshold {
			cb.state = CircuitOpen
		}
	}
}

// State возвращает текущее состояние (thread-safe read).
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats возвращает текущую статистику circuit breaker.
type Stats struct {
	State        CircuitState
	FailureCount int
}

// GetStats возвращает текущую статистику (thread-safe).
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:        cb.state,
		FailureCount: cb.failureCount,
	}
}
