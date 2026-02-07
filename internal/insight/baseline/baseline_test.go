package baseline

import (
	"math"
	"testing"
)

const epsilon = 1e-6

func TestEWMA_Convergence(t *testing.T) {
	tests := []struct {
		name     string
		alpha    float64
		value    float64
		samples  int
		wantMean float64
	}{
		{
			name:     "constant 100 converges",
			alpha:    0.1,
			value:    100.0,
			samples:  100,
			wantMean: 100.0,
		},
		{
			name:     "constant 42 converges",
			alpha:    0.2,
			value:    42.0,
			samples:  50,
			wantMean: 42.0,
		},
		{
			name:     "constant 0 converges",
			alpha:    0.1,
			value:    0.0,
			samples:  100,
			wantMean: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ewma := NewEWMA(tt.alpha)
			for i := 0; i < tt.samples; i++ {
				ewma.Update(tt.value)
			}
			if math.Abs(ewma.Mean-tt.wantMean) > epsilon {
				t.Errorf("Mean = %v, want %v", ewma.Mean, tt.wantMean)
			}
			// Variance should be near zero for constant values
			if ewma.StdDev() > 0.01 {
				t.Errorf("StdDev = %v, want near 0 for constant input", ewma.StdDev())
			}
		})
	}
}

func TestEWMA_ShiftDetection(t *testing.T) {
	tests := []struct {
		name         string
		alpha        float64
		initialValue float64
		initialCount int
		shiftValue   float64
		shiftCount   int
		tolerance    float64
	}{
		{
			name:         "shift from 50 to 100",
			alpha:        0.3,
			initialValue: 50.0,
			initialCount: 50,
			shiftValue:   100.0,
			shiftCount:   50,
			tolerance:    5.0,
		},
		{
			name:         "shift from 100 to 20",
			alpha:        0.2,
			initialValue: 100.0,
			initialCount: 30,
			shiftValue:   20.0,
			shiftCount:   50,
			tolerance:    10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ewma := NewEWMA(tt.alpha)
			// Feed initial values
			for i := 0; i < tt.initialCount; i++ {
				ewma.Update(tt.initialValue)
			}
			// Feed shift values
			for i := 0; i < tt.shiftCount; i++ {
				ewma.Update(tt.shiftValue)
			}
			// Mean should track toward shift value
			if math.Abs(ewma.Mean-tt.shiftValue) > tt.tolerance {
				t.Errorf("Mean = %v, want within %v of %v", ewma.Mean, tt.tolerance, tt.shiftValue)
			}
		})
	}
}

func TestEWMA_AlphaEffect(t *testing.T) {
	// Higher alpha should track faster after a shift
	slow := NewEWMA(0.1)
	fast := NewEWMA(0.5)

	// Both start at 50
	for i := 0; i < 20; i++ {
		slow.Update(50.0)
		fast.Update(50.0)
	}

	// Shift to 100
	for i := 0; i < 10; i++ {
		slow.Update(100.0)
		fast.Update(100.0)
	}

	// Fast should be closer to 100
	if fast.Mean <= slow.Mean {
		t.Errorf("Fast EWMA (alpha=0.5) should track faster than slow (alpha=0.1). fast=%v, slow=%v", fast.Mean, slow.Mean)
	}
}

func TestEWMA_StdDev(t *testing.T) {
	tests := []struct {
		name      string
		alpha     float64
		values    []float64
		wantStdDev float64
		tolerance  float64
	}{
		{
			name:       "low variance",
			alpha:      0.2,
			values:     []float64{10, 10.1, 9.9, 10.2, 9.8, 10.0, 10.1, 9.9},
			wantStdDev: 0.15,
			tolerance:  0.1,
		},
		{
			name:       "high variance",
			alpha:      0.2,
			values:     []float64{10, 20, 5, 25, 8, 18, 12, 22},
			wantStdDev: 7.0,
			tolerance:  3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ewma := NewEWMA(tt.alpha)
			for _, v := range tt.values {
				ewma.Update(v)
			}
			stdDev := ewma.StdDev()
			if math.Abs(stdDev-tt.wantStdDev) > tt.tolerance {
				t.Errorf("StdDev = %v, want %v +/- %v", stdDev, tt.wantStdDev, tt.tolerance)
			}
		})
	}
}

func TestEWMA_SingleSample(t *testing.T) {
	ewma := NewEWMA(0.1)
	ewma.Update(42.0)

	if ewma.Samples != 1 {
		t.Errorf("Samples = %v, want 1", ewma.Samples)
	}
	if math.Abs(ewma.Mean-42.0) > epsilon {
		t.Errorf("Mean = %v, want 42.0", ewma.Mean)
	}
	if ewma.StdDev() != 0.0 {
		t.Errorf("StdDev = %v, want 0 for single sample", ewma.StdDev())
	}
}

func TestEWMA_InvalidAlpha(t *testing.T) {
	tests := []struct {
		name      string
		alpha     float64
		wantAlpha float64
	}{
		{"zero alpha", 0.0, 0.1},
		{"negative alpha", -0.5, 0.1},
		{"alpha > 1", 1.5, 0.1},
		{"valid alpha", 0.3, 0.3},
		{"alpha = 1", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ewma := NewEWMA(tt.alpha)
			if math.Abs(ewma.Alpha-tt.wantAlpha) > epsilon {
				t.Errorf("Alpha = %v, want %v", ewma.Alpha, tt.wantAlpha)
			}
		})
	}
}

func TestHoltWinters_Seasonality(t *testing.T) {
	// Generate simple seasonal data: repeating pattern [10, 20, 30, 20]
	seasonLen := 4
	pattern := []float64{10, 20, 30, 20}
	hw := NewHoltWinters(0.3, 0.1, 0.3, seasonLen)

	// Feed 3 complete seasons
	for cycle := 0; cycle < 3; cycle++ {
		for _, v := range pattern {
			hw.Update(v)
		}
	}

	if !hw.IsInitialized() {
		t.Fatal("HoltWinters should be initialized after one season")
	}

	// Predict next 4 steps should match the pattern
	for i := 0; i < seasonLen; i++ {
		predicted := hw.Predict(i + 1)
		expected := pattern[i]
		tolerance := 5.0 // Loose tolerance for simple test
		if math.Abs(predicted-expected) > tolerance {
			t.Errorf("Predict(%d) = %v, want ~%v (tolerance %v)", i+1, predicted, expected, tolerance)
		}
	}
}

func TestHoltWinters_Trend(t *testing.T) {
	// Generate linearly increasing data: 0, 1, 2, 3, 4, ...
	seasonLen := 4
	hw := NewHoltWinters(0.3, 0.3, 0.1, seasonLen)

	for i := 0; i < 20; i++ {
		hw.Update(float64(i))
	}

	if !hw.IsInitialized() {
		t.Fatal("HoltWinters should be initialized")
	}

	// Trend should be positive
	if hw.Trend <= 0 {
		t.Errorf("Trend = %v, want > 0 for increasing data", hw.Trend)
	}

	// Level should be around the recent values
	if hw.Level < 10.0 || hw.Level > 25.0 {
		t.Errorf("Level = %v, want between 10 and 25", hw.Level)
	}
}

func TestHoltWinters_NotInitialized(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)

	// Feed fewer than seasonLen samples
	hw.Update(10)
	hw.Update(20)

	if hw.IsInitialized() {
		t.Error("HoltWinters should not be initialized before seasonLen samples")
	}

	predicted := hw.Predict(1)
	if predicted != 0 {
		t.Errorf("Predict before initialization = %v, want 0", predicted)
	}

	fitted := hw.Fitted()
	if fitted != 0 {
		t.Errorf("Fitted before initialization = %v, want 0", fitted)
	}
}

func TestHoltWinters_Fitted(t *testing.T) {
	// Simple seasonal pattern
	seasonLen := 4
	pattern := []float64{10, 20, 30, 20}
	hw := NewHoltWinters(0.3, 0.1, 0.3, seasonLen)

	// Feed 2 complete seasons
	for cycle := 0; cycle < 2; cycle++ {
		for _, v := range pattern {
			hw.Update(v)
		}
	}

	// Feed one more value and check fitted
	hw.Update(10.0)
	fitted := hw.Fitted()

	// Fitted should be reasonably close to the actual value
	tolerance := 5.0
	if math.Abs(fitted-10.0) > tolerance {
		t.Errorf("Fitted = %v, want ~10.0 (tolerance %v)", fitted, tolerance)
	}
}

func TestHoltWinters_SeasonLenValidation(t *testing.T) {
	tests := []struct {
		name           string
		seasonLen      int
		wantSeasonLen  int
	}{
		{"valid season length", 24, 24},
		{"zero season length", 0, 24},
		{"negative season length", -5, 24},
		{"one (too small)", 1, 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewHoltWinters(0.3, 0.1, 0.3, tt.seasonLen)
			if hw.SeasonLen != tt.wantSeasonLen {
				t.Errorf("SeasonLen = %v, want %v", hw.SeasonLen, tt.wantSeasonLen)
			}
		})
	}
}

func TestHoltWinters_ParameterClamping(t *testing.T) {
	tests := []struct {
		name  string
		alpha float64
		beta  float64
		gamma float64
	}{
		{"negative alpha", -0.5, 0.1, 0.1},
		{"alpha > 1", 1.5, 0.1, 0.1},
		{"negative beta", 0.3, -0.2, 0.1},
		{"beta > 1", 0.3, 2.0, 0.1},
		{"negative gamma", 0.3, 0.1, -0.8},
		{"gamma > 1", 0.3, 0.1, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewHoltWinters(tt.alpha, tt.beta, tt.gamma, 4)
			if hw.Alpha < 0 || hw.Alpha > 1 {
				t.Errorf("Alpha = %v, should be clamped to [0,1]", hw.Alpha)
			}
			if hw.Beta < 0 || hw.Beta > 1 {
				t.Errorf("Beta = %v, should be clamped to [0,1]", hw.Beta)
			}
			if hw.Gamma < 0 || hw.Gamma > 1 {
				t.Errorf("Gamma = %v, should be clamped to [0,1]", hw.Gamma)
			}
		})
	}
}

func BenchmarkEWMA_Update(b *testing.B) {
	ewma := NewEWMA(0.1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ewma.Update(float64(i % 100))
	}
}

func BenchmarkHoltWinters_Update(b *testing.B) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 24)
	// Initialize with one season
	for i := 0; i < 24; i++ {
		hw.Update(float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hw.Update(float64(i % 100))
	}
}
