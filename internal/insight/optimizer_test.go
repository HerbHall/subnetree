package insight

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestGenerateRecommendations_NilStore(t *testing.T) {
	o := NewOptimizer(nil, zap.NewNop())
	recs, err := o.GenerateRecommendations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations with nil store, got %d", len(recs))
	}
}
