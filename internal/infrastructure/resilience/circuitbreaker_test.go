// @sk-task T3.2: Unit-tests для CircuitBreaker (AC-003, AC-004)
// Проверка переходов состояний closed→open→half-open→closed.

package resilience

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	assert.Equal(t, CircuitClosed, cb.State())
	assert.Equal(t, 5, cb.threshold)
	assert.Equal(t, 30*time.Second, cb.timeout)
}

func TestNewCircuitBreaker_CustomConfig(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 3,
		Timeout:   5 * time.Second,
	}
	cb := NewCircuitBreaker(config)

	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_CanExecute_Closed(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	// В состоянии closed запросы разрешены
	err := cb.CanExecute()
	assert.NoError(t, err)
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 3,
		Timeout:   1 * time.Second,
	}
	cb := NewCircuitBreaker(config)

	// Начальное состояние — closed
	assert.Equal(t, CircuitClosed, cb.State())

	// Фиксируем 2 ошибки — всё ещё closed
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())

	// Третья ошибка — переход в open
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// В open запросы отклоняются
	err := cb.CanExecute()
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Переводим в open
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Ждём timeout
	time.Sleep(150 * time.Millisecond)

	// Теперь CanExecute должен разрешить запрос и перейти в half-open
	err := cb.CanExecute()
	assert.NoError(t, err)
	assert.Equal(t, CircuitHalfOpen, cb.State())
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Переводим в open, затем ждём и переходим в half-open
	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)
	cb.CanExecute() // переход в half-open
	assert.Equal(t, CircuitHalfOpen, cb.State())

	// Успешный запрос — возвращаемся в closed
	cb.RecordSuccess()
	assert.Equal(t, CircuitClosed, cb.State())

	// Теперь запросы разрешены
	err := cb.CanExecute()
	assert.NoError(t, err)
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Переводим в open, затем ждём и переходим в half-open
	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)
	cb.CanExecute()
	assert.Equal(t, CircuitHalfOpen, cb.State())

	// Неудачный запрос — снова в open
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Запросы снова отклоняются
	err := cb.CanExecute()
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_RecordSuccessInClosed(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 3,
		Timeout:   1 * time.Second,
	}
	cb := NewCircuitBreaker(config)

	// Фиксируем 2 ошибки
	cb.RecordFailure()
	cb.RecordFailure()
	stats := cb.GetStats()
	assert.Equal(t, 2, stats.FailureCount)

	// Успех сбрасывает счётчик
	cb.RecordSuccess()
	stats = cb.GetStats()
	assert.Equal(t, 0, stats.FailureCount)
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	stats := cb.GetStats()
	assert.Equal(t, CircuitClosed, stats.State)
	assert.Equal(t, 0, stats.FailureCount)

	// После ошибки
	cb.RecordFailure()
	stats = cb.GetStats()
	assert.Equal(t, CircuitClosed, stats.State)
	assert.Equal(t, 1, stats.FailureCount)
}

func TestCircuitState_String(t *testing.T) {
	assert.Equal(t, "closed", CircuitClosed.String())
	assert.Equal(t, "open", CircuitOpen.String())
	assert.Equal(t, "half-open", CircuitHalfOpen.String())
	assert.Equal(t, "unknown", CircuitState(99).String())
}

// @sk-task T2.2: тест параллельного probe в half-open (AC-003)
func TestCircuitBreaker_HalfOpen_ParallelProbe(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Переводим в open
	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)

	const n = 10
	var allowed atomic.Int64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			runtime.Gosched()
			if cb.CanExecute() == nil {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), allowed.Load(), "ровно одна горутина должна получить probe-разрешение")
}

// @sk-task T2.2: тест сброса probe-флага между циклами (AC-004)
func TestCircuitBreaker_HalfOpen_ProbeReset(t *testing.T) {
	config := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	for cycle := 0; cycle < 2; cycle++ {
		// Переводим в open
		cb.RecordFailure()
		time.Sleep(150 * time.Millisecond)

		const n = 10
		var allowed atomic.Int64
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				runtime.Gosched()
				if cb.CanExecute() == nil {
					allowed.Add(1)
				}
			}()
		}
		wg.Wait()

		assert.Equal(t, int64(1), allowed.Load(), "цикл %d: ровно одна горутина должна получить probe-разрешение", cycle+1)

		// Сбрасываем через неудачный probe (возвращаемся в open для следующего цикла)
		cb.RecordFailure()
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		Threshold: 10,
		Timeout:   1 * time.Second,
	})

	// Конкурентные запросы на чтение состояния
	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func() {
			_ = cb.CanExecute()
			_ = cb.State()
			_ = cb.GetStats()
			done <- true
		}()
	}

	// Конкурентные запросы на запись
	for i := 0; i < 50; i++ {
		go func() {
			cb.RecordFailure()
			cb.RecordSuccess()
			done <- true
		}()
	}

	// Ждём завершения
	for i := 0; i < 100; i++ {
		<-done
	}

	// Не должно быть panic или deadlock
	assert.True(t, true)
}
