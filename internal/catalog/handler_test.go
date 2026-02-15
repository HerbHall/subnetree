package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgcatalog "github.com/HerbHall/subnetree/pkg/catalog"
	"go.uber.org/zap"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	cat := pkgcatalog.NewCatalog()
	engine := NewEngine(cat)
	return NewHandler(engine, zap.NewNop())
}

func TestHandleRecommendations_DefaultTier(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/recommendations", http.NoBody)
	rec := httptest.NewRecorder()

	h.handleRecommendations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp RecommendationResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Tier != 1 {
		t.Errorf("expected default tier 1, got %d", resp.Tier)
	}
	if resp.Count == 0 {
		t.Error("expected non-zero count for default tier")
	}
	if resp.Count != len(resp.Entries) {
		t.Errorf("count (%d) does not match entries length (%d)", resp.Count, len(resp.Entries))
	}
}

func TestHandleRecommendations_SpecificTier(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/recommendations?tier=0", http.NoBody)
	rec := httptest.NewRecorder()

	h.handleRecommendations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp RecommendationResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Tier != 0 {
		t.Errorf("expected tier 0, got %d", resp.Tier)
	}

	// Tier 0 should have fewer entries than tier 1 (some tools excluded).
	h2 := newTestHandler(t)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/recommendations?tier=1", http.NoBody)
	rec2 := httptest.NewRecorder()
	h2.handleRecommendations(rec2, req2)

	var resp2 RecommendationResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode tier 1 response: %v", err)
	}

	if resp.Count >= resp2.Count {
		t.Errorf("tier 0 (%d entries) should have fewer entries than tier 1 (%d entries)",
			resp.Count, resp2.Count)
	}
}

func TestHandleRecommendations_InvalidTier(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "negative", query: "?tier=-1"},
		{name: "too high", query: "?tier=5"},
		{name: "non-numeric", query: "?tier=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/recommendations"+tt.query, http.NoBody)
			rec := httptest.NewRecorder()

			h.handleRecommendations(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rec.Code)
			}
		})
	}
}

func TestHandleRecommendations_CategoryFilter(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/recommendations?tier=1&category=monitoring", http.NoBody)
	rec := httptest.NewRecorder()

	h.handleRecommendations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp RecommendationResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Count == 0 {
		t.Fatal("expected monitoring entries for tier 1")
	}
	for i := range resp.Entries {
		if resp.Entries[i].Category != pkgcatalog.CategoryMonitoring {
			t.Errorf("entry %s has category %s, expected monitoring",
				resp.Entries[i].Name, resp.Entries[i].Category)
		}
	}
}

func TestHandleListEntries(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/entries", http.NoBody)
	rec := httptest.NewRecorder()

	h.handleListEntries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var entries []pkgcatalog.CatalogEntry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 48 {
		t.Errorf("expected 48 entries, got %d", len(entries))
	}
}
