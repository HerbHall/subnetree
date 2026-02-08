package anomaly

import (
	"math"
	"testing"
)

func TestZScoreCheck_NormalValue(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		mean      float64
		stdDev    float64
		threshold float64
		wantZ     float64
	}{
		{
			name:      "value at mean",
			value:     100.0,
			mean:      100.0,
			stdDev:    10.0,
			threshold: 3.0,
			wantZ:     0.0,
		},
		{
			name:      "value within 1 stddev",
			value:     105.0,
			mean:      100.0,
			stdDev:    10.0,
			threshold: 3.0,
			wantZ:     0.5,
		},
		{
			name:      "value at threshold boundary",
			value:     129.9,
			mean:      100.0,
			stdDev:    10.0,
			threshold: 3.0,
			wantZ:     2.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ZScoreCheck(tt.value, tt.mean, tt.stdDev, tt.threshold)
			if result.IsAnomaly {
				t.Errorf("ZScoreCheck() IsAnomaly = true, want false")
			}
			if math.Abs(result.ZScore-tt.wantZ) > 0.01 {
				t.Errorf("ZScoreCheck() ZScore = %v, want %v", result.ZScore, tt.wantZ)
			}
			if result.Severity != "" {
				t.Errorf("ZScoreCheck() Severity = %v, want empty", result.Severity)
			}
		})
	}
}

func TestZScoreCheck_AnomalousValue(t *testing.T) {
	tests := []struct {
		name          string
		value         float64
		mean          float64
		stdDev        float64
		threshold     float64
		wantAnomaly   bool
		wantZApprox   float64
		wantSeverity  string
	}{
		{
			name:         "value at threshold",
			value:        130.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  3.0,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "value beyond threshold",
			value:        135.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  3.5,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "value at critical threshold",
			value:        140.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  4.0,
			wantSeverity: SeverityCritical,
		},
		{
			name:         "value far beyond threshold",
			value:        150.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  5.0,
			wantSeverity: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ZScoreCheck(tt.value, tt.mean, tt.stdDev, tt.threshold)
			if result.IsAnomaly != tt.wantAnomaly {
				t.Errorf("ZScoreCheck() IsAnomaly = %v, want %v", result.IsAnomaly, tt.wantAnomaly)
			}
			if math.Abs(result.ZScore-tt.wantZApprox) > 0.01 {
				t.Errorf("ZScoreCheck() ZScore = %v, want %v", result.ZScore, tt.wantZApprox)
			}
			if result.Severity != tt.wantSeverity {
				t.Errorf("ZScoreCheck() Severity = %v, want %v", result.Severity, tt.wantSeverity)
			}
		})
	}
}

func TestZScoreCheck_Severity(t *testing.T) {
	tests := []struct {
		name         string
		zScoreMult   float64
		threshold    float64
		wantSeverity string
	}{
		{
			name:         "exactly at threshold is warning",
			zScoreMult:   3.0,
			threshold:    3.0,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "slightly above threshold is warning",
			zScoreMult:   3.5,
			threshold:    3.0,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "just below critical is warning",
			zScoreMult:   3.99,
			threshold:    3.0,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "exactly at critical threshold",
			zScoreMult:   4.0,
			threshold:    3.0,
			wantSeverity: SeverityCritical,
		},
		{
			name:         "far beyond critical",
			zScoreMult:   10.0,
			threshold:    3.0,
			wantSeverity: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mean := 100.0
			stdDev := 10.0
			value := mean + tt.zScoreMult*stdDev

			result := ZScoreCheck(value, mean, stdDev, tt.threshold)
			if !result.IsAnomaly {
				t.Errorf("ZScoreCheck() IsAnomaly = false, want true")
			}
			if result.Severity != tt.wantSeverity {
				t.Errorf("ZScoreCheck() Severity = %v, want %v", result.Severity, tt.wantSeverity)
			}
		})
	}
}

func TestZScoreCheck_ZeroStdDev(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		mean   float64
		stdDev float64
	}{
		{
			name:   "zero stddev",
			value:  100.0,
			mean:   100.0,
			stdDev: 0.0,
		},
		{
			name:   "negative stddev",
			value:  100.0,
			mean:   100.0,
			stdDev: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ZScoreCheck(tt.value, tt.mean, tt.stdDev, 3.0)
			if result.IsAnomaly {
				t.Errorf("ZScoreCheck() IsAnomaly = true, want false for invalid stddev")
			}
			if result.ZScore != 0.0 {
				t.Errorf("ZScoreCheck() ZScore = %v, want 0.0", result.ZScore)
			}
			if result.Severity != "" {
				t.Errorf("ZScoreCheck() Severity = %v, want empty", result.Severity)
			}
		})
	}
}

func TestZScoreCheck_NegativeDeviation(t *testing.T) {
	tests := []struct {
		name         string
		value        float64
		mean         float64
		stdDev       float64
		threshold    float64
		wantAnomaly  bool
		wantZApprox  float64
		wantSeverity string
	}{
		{
			name:         "negative z-score at threshold",
			value:        70.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  -3.0,
			wantSeverity: SeverityWarning,
		},
		{
			name:         "negative z-score critical",
			value:        60.0,
			mean:         100.0,
			stdDev:       10.0,
			threshold:    3.0,
			wantAnomaly:  true,
			wantZApprox:  -4.0,
			wantSeverity: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ZScoreCheck(tt.value, tt.mean, tt.stdDev, tt.threshold)
			if result.IsAnomaly != tt.wantAnomaly {
				t.Errorf("ZScoreCheck() IsAnomaly = %v, want %v", result.IsAnomaly, tt.wantAnomaly)
			}
			if math.Abs(result.ZScore-tt.wantZApprox) > 0.01 {
				t.Errorf("ZScoreCheck() ZScore = %v, want %v", result.ZScore, tt.wantZApprox)
			}
			if result.Severity != tt.wantSeverity {
				t.Errorf("ZScoreCheck() Severity = %v, want %v", result.Severity, tt.wantSeverity)
			}
		})
	}
}

func TestCUSUM_NoChange(t *testing.T) {
	cusum := NewCUSUM(0.5, 5.0)

	// Feed normalized values around zero (stable mean)
	for i := 0; i < 10; i++ {
		normalized := 0.1 // Small positive value within drift
		result := cusum.Update(normalized)
		if result.IsChangePoint {
			t.Errorf("CUSUM.Update() detected change point at iteration %d, want none", i)
		}
	}
}

func TestCUSUM_StepChangeUp(t *testing.T) {
	cusum := NewCUSUM(0.5, 5.0)

	// Persistent positive shift (mean increased by 1 stddev)
	changeDetected := false
	for i := 0; i < 20; i++ {
		normalized := 1.0 // Persistent positive deviation
		result := cusum.Update(normalized)
		if result.IsChangePoint {
			changeDetected = true
			if result.Direction != "up" {
				t.Errorf("CUSUM.Update() Direction = %v, want up", result.Direction)
			}
			break
		}
	}

	if !changeDetected {
		t.Errorf("CUSUM.Update() did not detect upward change point")
	}
}

func TestCUSUM_StepChangeDown(t *testing.T) {
	cusum := NewCUSUM(0.5, 5.0)

	// Persistent negative shift (mean decreased by 1 stddev)
	changeDetected := false
	for i := 0; i < 20; i++ {
		normalized := -1.0 // Persistent negative deviation
		result := cusum.Update(normalized)
		if result.IsChangePoint {
			changeDetected = true
			if result.Direction != "down" {
				t.Errorf("CUSUM.Update() Direction = %v, want down", result.Direction)
			}
			break
		}
	}

	if !changeDetected {
		t.Errorf("CUSUM.Update() did not detect downward change point")
	}
}

func TestCUSUM_Drift(t *testing.T) {
	cusum := NewCUSUM(0.5, 5.0)

	// Small fluctuations within drift tolerance should not trigger
	fluctuations := []float64{0.3, -0.2, 0.4, -0.1, 0.2, -0.3, 0.1}
	for i, v := range fluctuations {
		result := cusum.Update(v)
		if result.IsChangePoint {
			t.Errorf("CUSUM.Update() detected change point at iteration %d with drift-acceptable fluctuation", i)
		}
	}
}

func TestCUSUM_Reset(t *testing.T) {
	cusum := NewCUSUM(0.5, 5.0)

	// Build up some cumulative sum
	for i := 0; i < 5; i++ {
		cusum.Update(0.8)
	}

	if cusum.High == 0.0 {
		t.Error("CUSUM.High should be non-zero after updates")
	}

	cusum.Reset()

	if cusum.High != 0.0 {
		t.Errorf("CUSUM.High = %v after Reset(), want 0.0", cusum.High)
	}
	if cusum.Low != 0.0 {
		t.Errorf("CUSUM.Low = %v after Reset(), want 0.0", cusum.Low)
	}
}
