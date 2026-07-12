// @sk-test health-check-interface#T4.1: unit tests for HealthChecker + HTTP handlers (AC-005, AC-006, AC-007)
package draftrag

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// @sk-test health-check-interface#T4.1: empty component list → healthy
func TestNewHealthChecker_NilComponents(t *testing.T) {
	hc := NewHealthChecker()
	result := hc.Check(context.Background())
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Healthy {
		t.Fatal("expected healthy with no components")
	}
}

// @sk-test health-check-interface#T4.1: all components healthy
func TestNewHealthChecker_AllHealthy(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "store", Health: func(_ context.Context) error { return nil }},
		ComponentHealth{Name: "embedder", Health: func(_ context.Context) error { return nil }},
	)
	result := hc.Check(context.Background())
	if !result.Healthy {
		t.Fatalf("expected healthy, got: %v", result.Error)
	}
}

// @sk-test health-check-interface#T4.1: HealthChecker aggregates errors (AC-005)
func TestHealthChecker_AggregatesErrors(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "ok", Health: func(_ context.Context) error { return nil }},
		ComponentHealth{Name: "fail1", Health: func(_ context.Context) error { return errors.New("err1") }},
		ComponentHealth{Name: "fail2", Health: func(_ context.Context) error { return errors.New("err2") }},
	)
	result := hc.Check(context.Background())
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
	if result.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

// @sk-test health-check-interface#T4.1: context.Canceled → unhealthy
func TestHealthChecker_ContextCancellation(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "slow", Health: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := hc.Check(ctx)
	if result.Healthy {
		t.Fatal("expected unhealthy after cancellation")
	}
	if result.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

// @sk-test health-check-interface#T4.1: deadline exceeded → unhealthy
func TestHealthChecker_ContextDeadline(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "slow", Health: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)
	result := hc.Check(ctx)
	if result.Healthy {
		t.Fatal("expected unhealthy after deadline")
	}
}

// @sk-test health-check-interface#T4.1: concurrent Register + Check races
func TestHealthChecker_ConcurrentSafety(_ *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "a", Health: func(_ context.Context) error { return nil }},
		ComponentHealth{Name: "b", Health: func(_ context.Context) error { return nil }},
	)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			hc.Check(context.Background())
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		hc.Register(ComponentHealth{Name: "c", Health: func(_ context.Context) error { return nil }})
	}
	<-done
}

// @sk-test health-check-interface#T4.1: Register adds component dynamically
func TestHealthChecker_Register(t *testing.T) {
	hc := NewHealthChecker()
	var called atomic.Bool
	hc.Register(ComponentHealth{Name: "later", Health: func(_ context.Context) error {
		called.Store(true)
		return nil
	}})
	result := hc.Check(context.Background())
	if !result.Healthy {
		t.Fatalf("expected healthy, got: %v", result.Error)
	}
	if !called.Load() {
		t.Fatal("expected registered component to be checked")
	}
}

// @sk-test health-check-interface#T4.1: nil receiver panics
func TestHealthChecker_NilReceiverPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil receiver")
		}
	}()
	var hc *HealthChecker
	hc.Check(context.Background())
}

// @sk-test health-check-interface#T4.1: nil context panics
func TestHealthChecker_NilContextPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	hc := NewHealthChecker()
	hc.Check(nil) //nolint:staticcheck // intentional: test panic on nil
}

// @sk-test health-check-interface#T4.1: LivenessHandler always 200 (AC-006)
func TestLivenessHandler_Always200(t *testing.T) {
	h := LivenessHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Fatalf("expected 'OK', got %q", w.Body.String())
	}
}

// @sk-test health-check-interface#T4.1: ReadinessHandler 200 when all healthy (AC-007)
func TestReadinessHandler_AllHealthy(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "store", Health: func(_ context.Context) error { return nil }},
	)
	h := ReadinessHandler(hc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result HealthCheckerResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy, got: %v", result.Error)
	}
}

// @sk-test health-check-interface#T4.1: ReadinessHandler 503 when unhealthy (AC-007)
func TestReadinessHandler_Unhealthy_503(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "store", Health: func(_ context.Context) error { return errors.New("down") }},
	)
	h := ReadinessHandler(hc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	h(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var result HealthCheckerResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected unhealthy")
	}
}

// @sk-test health-check-interface#T4.1: nil HealthChecker panics
func TestReadinessHandler_NilCheckerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil HealthChecker")
		}
	}()
	ReadinessHandler(nil)
}

// @sk-test health-check-interface#T4.1: StartupHandler behaviour matches ReadinessHandler
func TestStartupHandler_EquivalentToReadiness(t *testing.T) {
	hc := NewHealthChecker(
		ComponentHealth{Name: "store", Health: func(_ context.Context) error { return nil }},
	)
	s := StartupHandler(hc)
	r := ReadinessHandler(hc)

	sw := httptest.NewRecorder()
	sr := httptest.NewRequest(http.MethodGet, "/startupz", nil)
	s(sw, sr)

	rw := httptest.NewRecorder()
	rr := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	r(rw, rr)

	if sw.Code != rw.Code {
		t.Fatalf("expected same status, got startup=%d readiness=%d", sw.Code, rw.Code)
	}
	if sw.Body.String() != rw.Body.String() {
		t.Fatalf("expected same body, got startup=%q readiness=%q", sw.Body.String(), rw.Body.String())
	}
}

// @sk-test health-check-interface#T4.1: nil HealthChecker panics in StartupHandler
func TestStartupHandler_NilCheckerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil HealthChecker")
		}
	}()
	StartupHandler(nil)
}
