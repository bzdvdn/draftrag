package draftrag

import (
	"context"
	"testing"
)

// @sk-test arch-issues#T5.4: generated maps have correct entry count (AC-005, AC-006)

// expectedRouteCount is the number of route constants defined.
// Must match the constant block in search_routing.go.
const expectedRouteCount = 9

func TestGeneratedMaps_HaveCorrectEntryCount(t *testing.T) {
	if got := len(retrieveHandlers); got != expectedRouteCount {
		t.Errorf("retrieveHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(answerHandlers); got != expectedRouteCount {
		t.Errorf("answerHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(citeHandlers); got != expectedRouteCount {
		t.Errorf("citeHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(inlineCiteHandlers); got != expectedRouteCount {
		t.Errorf("inlineCiteHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(streamHandlers); got != expectedRouteCount {
		t.Errorf("streamHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(streamSourcesHandlers); got != expectedRouteCount {
		t.Errorf("streamSourcesHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
	if got := len(streamCiteHandlers); got != expectedRouteCount {
		t.Errorf("streamCiteHandlers: expected %d entries, got %d", expectedRouteCount, got)
	}
}

// @sk-test arch-issues#T5.4: all routes present in every generated map (AC-005, AC-006)
func TestGeneratedMaps_AllRoutesPresent(t *testing.T) {
	routes := []route{routeBasic, routeRewriter, routeSubDecompose, routeHyDE,
		routeMultiQuery, routeTools, routeHybrid, routeParentIDs, routeFilter}

	for _, rc := range routes {
		if _, ok := retrieveHandlers[rc]; !ok {
			t.Errorf("retrieveHandlers missing route %d", rc)
		}
		if _, ok := answerHandlers[rc]; !ok {
			t.Errorf("answerHandlers missing route %d", rc)
		}
		if _, ok := citeHandlers[rc]; !ok {
			t.Errorf("citeHandlers missing route %d", rc)
		}
		if _, ok := inlineCiteHandlers[rc]; !ok {
			t.Errorf("inlineCiteHandlers missing route %d", rc)
		}
		if _, ok := streamHandlers[rc]; !ok {
			t.Errorf("streamHandlers missing route %d", rc)
		}
		if _, ok := streamSourcesHandlers[rc]; !ok {
			t.Errorf("streamSourcesHandlers missing route %d", rc)
		}
		if _, ok := streamCiteHandlers[rc]; !ok {
			t.Errorf("streamCiteHandlers missing route %d", rc)
		}
	}
}

// @sk-test arch-issues#T5.4: generated handler can be invoked via router (AC-005, AC-006)
func TestGeneratedMaps_RouterIntegration(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx := context.Background()

	// Route basic via retrieve router
	res, err := retrieveRouter.execute(ctx, "concurrency", 2, routeBasic, p.Search("test"))
	if err != nil {
		t.Fatalf("basic retrieve via generated map: %v", err)
	}
	if len(res.Result.Chunks) == 0 {
		t.Fatal("expected results from generated map handler")
	}
}
