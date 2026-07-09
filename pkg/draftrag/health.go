// @sk-task health-check-interface#T2.1: HealthChecker + HTTP handlers (AC-005, AC-006, AC-007, RQ-002, RQ-003, RQ-009)
package draftrag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// HealthCheckFunc — функция проверки здоровья компонента.
type HealthCheckFunc func(context.Context) error

// ComponentHealth описывает компонент с именем и функцией проверки.
type ComponentHealth struct {
	Name   string
	Health HealthCheckFunc
}

// HealthCheckerResult содержит результат проверки одного компонента.
type HealthCheckerResult struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// HealthChecker проверяет доступность зарегистрированных компонентов.
type HealthChecker struct {
	mu         sync.RWMutex
	components []ComponentHealth
}

// NewHealthChecker создаёт HealthChecker с указанными компонентами.
// Паникует при nil *HealthChecker — programmer error.
func NewHealthChecker(components ...ComponentHealth) *HealthChecker {
	return &HealthChecker{
		components: components,
	}
}

// Register добавляет компонент для проверки.
func (hc *HealthChecker) Register(component ComponentHealth) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.components = append(hc.components, component)
}

// Check проверяет все зарегистрированные компоненты.
// Если хотя бы один компонент вернул ошибку — возвращает aggregated error с именами проблемных компонентов.
// Если контекст истёк — возвращает context.DeadlineExceeded / context.Canceled.
// Если компонентов нет (nil/empty slice) — всегда nil.
func (hc *HealthChecker) Check(ctx context.Context) *HealthCheckerResult {
	if hc == nil {
		panic("nil HealthChecker")
	}
	if ctx == nil {
		panic("nil context")
	}

	hc.mu.RLock()
	components := hc.components
	hc.mu.RUnlock()

	if len(components) == 0 {
		return &HealthCheckerResult{Name: "HealthChecker", Healthy: true}
	}

	var (
		unhealthy []string
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	for _, comp := range components {
		wg.Add(1)
		comp := comp
		go func() {
			defer wg.Done()
			err := comp.Health(ctx)
			if err != nil {
				mu.Lock()
				unhealthy = append(unhealthy, comp.Name)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return &HealthCheckerResult{Name: "HealthChecker", Healthy: false, Error: ctx.Err().Error()}
	}

	if len(unhealthy) > 0 {
		errMsg := fmt.Sprintf("unhealthy components: %v", unhealthy)
		return &HealthCheckerResult{Name: "HealthChecker", Healthy: false, Error: errMsg}
	}

	return &HealthCheckerResult{Name: "HealthChecker", Healthy: true}
}

// LivenessHandler возвращает http.HandlerFunc, который всегда отвечает 200 OK.
// Не проверяет зависимости — только процесс жив.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// ReadinessHandler возвращает http.HandlerFunc, который проверяет компоненты через HealthChecker.
// 200 OK если все здоровы, 503 Service Unavailable + JSON с ошибками если нет.
func ReadinessHandler(hc *HealthChecker) http.HandlerFunc {
	if hc == nil {
		panic("nil HealthChecker")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		result := hc.Check(r.Context())
		writeHealthResponse(w, result)
	}
}

// StartupHandler возвращает http.HandlerFunc, идентичный ReadinessHandler.
// Разделение Startup/Readiness позволяет пользователю настроить разную частоту проверок в K8s.
func StartupHandler(hc *HealthChecker) http.HandlerFunc {
	if hc == nil {
		panic("nil HealthChecker")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		result := hc.Check(r.Context())
		writeHealthResponse(w, result)
	}
}

func writeHealthResponse(w http.ResponseWriter, result *HealthCheckerResult) {
	body, err := json.Marshal(result)
	if err != nil {
		http.Error(w, `{"error":"marshal error"}`, http.StatusInternalServerError)
		return
	}

	if result.Healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}


