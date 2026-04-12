package otel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestHooks_StageEnd_EmitsMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	hooks, err := NewHooks(HooksOptions{
		TracerProvider: noop.NewTracerProvider(),
		MeterProvider:  mp,
	})
	require.NoError(t, err)

	ctx := context.Background()
	hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: "Query",
		Stage:     domain.HookStageSearch,
		Duration:  123 * time.Millisecond,
		Err:       nil,
	})
	hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: "Query",
		Stage:     domain.HookStageSearch,
		Duration:  200 * time.Millisecond,
		Err:       errors.New("fail"),
	})

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	var (
		foundDuration bool
		foundErrors   bool
	)

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case MetricStageDurationMS:
				foundDuration = true
				_, ok := m.Data.(metricdata.Histogram[float64])
				require.True(t, ok, "duration metric must be histogram")
			case MetricStageErrors:
				foundErrors = true
				_, ok := m.Data.(metricdata.Sum[int64])
				require.True(t, ok, "errors metric must be sum")
			}
		}
	}

	require.True(t, foundDuration, "duration metric not found")
	require.True(t, foundErrors, "errors metric not found")
}
