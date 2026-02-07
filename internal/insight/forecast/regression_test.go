package forecast

import (
	"math"
	"testing"
	"time"
)

func TestLinearRegression_PerfectLine(t *testing.T) {
	t.Parallel()

	// y = 2x + 1
	times := []float64{0, 1, 2, 3, 4}
	values := []float64{1, 3, 5, 7, 9}
	threshold := 15.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Check slope (should be exactly 2)
	if math.Abs(result.Slope-2.0) > 0.0001 {
		t.Errorf("Slope = %v, want 2.0", result.Slope)
	}

	// Check intercept (should be exactly 1)
	if math.Abs(result.Intercept-1.0) > 0.0001 {
		t.Errorf("Intercept = %v, want 1.0", result.Intercept)
	}

	// R-squared should be 1.0 for perfect linear fit
	if math.Abs(result.RSquared-1.0) > 0.0001 {
		t.Errorf("RSquared = %v, want 1.0", result.RSquared)
	}

	// Predicted value at t=4 should be 9
	if math.Abs(result.Predicted-9.0) > 0.0001 {
		t.Errorf("Predicted = %v, want 9.0", result.Predicted)
	}

	// Time to limit: (15 - 9) / 2 = 3 hours
	if result.TimeToLimit == nil {
		t.Fatal("expected TimeToLimit, got nil")
	}
	expected := 3 * time.Hour
	if math.Abs(result.TimeToLimit.Hours()-expected.Hours()) > 0.01 {
		t.Errorf("TimeToLimit = %v, want %v", result.TimeToLimit, expected)
	}
}

func TestLinearRegression_FlatLine(t *testing.T) {
	t.Parallel()

	times := []float64{0, 1, 2, 3, 4}
	values := []float64{5, 5, 5, 5, 5}
	threshold := 10.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Slope should be 0
	if math.Abs(result.Slope) > 0.0001 {
		t.Errorf("Slope = %v, want 0", result.Slope)
	}

	// Intercept should be mean value (5)
	if math.Abs(result.Intercept-5.0) > 0.0001 {
		t.Errorf("Intercept = %v, want 5.0", result.Intercept)
	}

	// Predicted should be 5
	if math.Abs(result.Predicted-5.0) > 0.0001 {
		t.Errorf("Predicted = %v, want 5.0", result.Predicted)
	}

	// No time to limit since not approaching threshold
	if result.TimeToLimit != nil {
		t.Errorf("TimeToLimit should be nil for flat line, got %v", result.TimeToLimit)
	}
}

func TestLinearRegression_NoisyData(t *testing.T) {
	t.Parallel()

	// Underlying trend: y = 1.5x + 2, with noise
	times := []float64{0, 1, 2, 3, 4, 5, 6}
	values := []float64{2.1, 3.4, 5.2, 6.8, 8.1, 9.6, 11.0}
	threshold := 20.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Slope should be approximately 1.5
	if math.Abs(result.Slope-1.5) > 0.2 {
		t.Errorf("Slope = %v, want approximately 1.5", result.Slope)
	}

	// Intercept should be approximately 2
	if math.Abs(result.Intercept-2.0) > 0.5 {
		t.Errorf("Intercept = %v, want approximately 2.0", result.Intercept)
	}

	// R-squared should be high but not perfect
	if result.RSquared < 0.95 || result.RSquared > 1.0 {
		t.Errorf("RSquared = %v, want between 0.95 and 1.0", result.RSquared)
	}

	// Should have time to limit
	if result.TimeToLimit == nil {
		t.Error("expected TimeToLimit for rising trend, got nil")
	}
}

func TestLinearRegression_TimeToLimit(t *testing.T) {
	t.Parallel()

	// Rising data: y = 3x + 10
	times := []float64{0, 1, 2, 3}
	values := []float64{10, 13, 16, 19}
	threshold := 25.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// At t=3, predicted=19. To reach 25: (25-19)/3 = 2 hours
	if result.TimeToLimit == nil {
		t.Fatal("expected TimeToLimit, got nil")
	}
	expectedHours := 2.0
	if math.Abs(result.TimeToLimit.Hours()-expectedHours) > 0.1 {
		t.Errorf("TimeToLimit = %v hours, want approximately %v hours",
			result.TimeToLimit.Hours(), expectedHours)
	}
}

func TestLinearRegression_DecreasingToLimit(t *testing.T) {
	t.Parallel()

	// Decreasing data: y = -2x + 20
	times := []float64{0, 1, 2, 3}
	values := []float64{20, 18, 16, 14}
	threshold := 10.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// At t=3, predicted=14. To reach 10: (10-14)/(-2) = 2 hours
	if result.TimeToLimit == nil {
		t.Fatal("expected TimeToLimit for decreasing trend, got nil")
	}
	expectedHours := 2.0
	if math.Abs(result.TimeToLimit.Hours()-expectedHours) > 0.1 {
		t.Errorf("TimeToLimit = %v hours, want approximately %v hours",
			result.TimeToLimit.Hours(), expectedHours)
	}
}

func TestLinearRegression_NoTimeToLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		times     []float64
		values    []float64
		threshold float64
		reason    string
	}{
		{
			name:      "flat line",
			times:     []float64{0, 1, 2, 3},
			values:    []float64{5, 5, 5, 5},
			threshold: 10.0,
			reason:    "slope is zero",
		},
		{
			name:      "rising but already above threshold",
			times:     []float64{0, 1, 2, 3},
			values:    []float64{10, 12, 14, 16},
			threshold: 15.0,
			reason:    "already at or above threshold",
		},
		{
			name:      "decreasing away from threshold",
			times:     []float64{0, 1, 2, 3},
			values:    []float64{20, 18, 16, 14},
			threshold: 25.0,
			reason:    "decreasing but threshold is above",
		},
		{
			name:      "rising away from upper threshold",
			times:     []float64{0, 1, 2, 3},
			values:    []float64{2, 4, 6, 8},
			threshold: 5.0,
			reason:    "rising but threshold is below",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LinearRegression(tt.times, tt.values, tt.threshold)
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.TimeToLimit != nil {
				t.Errorf("expected no TimeToLimit (%s), got %v", tt.reason, result.TimeToLimit)
			}
		})
	}
}

func TestLinearRegression_TooFewPoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		times  []float64
		values []float64
	}{
		{
			name:   "empty",
			times:  []float64{},
			values: []float64{},
		},
		{
			name:   "one point",
			times:  []float64{0},
			values: []float64{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LinearRegression(tt.times, tt.values, 10.0)
			if result != nil {
				t.Errorf("expected nil for %s, got %+v", tt.name, result)
			}
		})
	}
}

func TestLinearRegression_MismatchedLengths(t *testing.T) {
	t.Parallel()

	times := []float64{0, 1, 2}
	values := []float64{1, 3}

	result := LinearRegression(times, values, 10.0)
	if result != nil {
		t.Errorf("expected nil for mismatched lengths, got %+v", result)
	}
}

func TestTimeToHours(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	timestamps := []time.Time{
		base,
		base.Add(1 * time.Hour),
		base.Add(2 * time.Hour),
		base.Add(3*time.Hour + 30*time.Minute),
	}

	hours := TimeToHours(timestamps)

	expected := []float64{0, 1, 2, 3.5}
	if len(hours) != len(expected) {
		t.Fatalf("length = %d, want %d", len(hours), len(expected))
	}

	for i := range expected {
		if math.Abs(hours[i]-expected[i]) > 0.0001 {
			t.Errorf("hours[%d] = %v, want %v", i, hours[i], expected[i])
		}
	}
}

func TestTimeToHours_Empty(t *testing.T) {
	t.Parallel()

	result := TimeToHours([]time.Time{})
	if result != nil {
		t.Errorf("expected nil for empty timestamps, got %v", result)
	}
}

func TestLinearRegression_ZeroVarianceX(t *testing.T) {
	t.Parallel()

	// All times are the same (no variance in X)
	times := []float64{5, 5, 5, 5}
	values := []float64{10, 12, 11, 13}
	threshold := 20.0

	result := LinearRegression(times, values, threshold)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Slope should be 0 when X has no variance
	if result.Slope != 0 {
		t.Errorf("Slope = %v, want 0 for zero variance in X", result.Slope)
	}

	// Intercept should be mean of Y values
	meanY := (10 + 12 + 11 + 13) / 4.0
	if math.Abs(result.Intercept-meanY) > 0.0001 {
		t.Errorf("Intercept = %v, want %v (mean of Y)", result.Intercept, meanY)
	}

	// Predicted should equal intercept
	if result.Predicted != result.Intercept {
		t.Errorf("Predicted = %v, want %v (equal to intercept)", result.Predicted, result.Intercept)
	}
}
