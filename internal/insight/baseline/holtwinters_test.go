package baseline

import (
	"math"
	"testing"
)

func TestHoltWinters_Forecast(t *testing.T) {
	tests := []struct {
		name       string
		pattern    []float64
		seasonLen  int
		cycles     int
		steps      int
		tolerance  float64
	}{
		{
			name:      "daily pattern forecast 4 steps",
			pattern:   []float64{10, 20, 30, 20},
			seasonLen: 4,
			cycles:    5,
			steps:     4,
			tolerance: 5.0,
		},
		{
			name:      "weekly-like pattern forecast 7 steps",
			pattern:   []float64{5, 10, 15, 20, 15, 10, 5},
			seasonLen: 7,
			cycles:    4,
			steps:     7,
			tolerance: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewHoltWinters(0.3, 0.1, 0.3, tt.seasonLen)

			for cycle := 0; cycle < tt.cycles; cycle++ {
				for _, v := range tt.pattern {
					hw.Update(v)
				}
			}

			forecast := hw.Forecast(tt.steps)
			if forecast == nil {
				t.Fatal("Forecast returned nil for initialized model")
			}
			if len(forecast) != tt.steps {
				t.Fatalf("Forecast returned %d steps, want %d", len(forecast), tt.steps)
			}

			for i, predicted := range forecast {
				expected := tt.pattern[i%tt.seasonLen]
				if math.Abs(predicted-expected) > tt.tolerance {
					t.Errorf("Forecast[%d] = %.2f, want ~%.2f (tolerance %.2f)",
						i, predicted, expected, tt.tolerance)
				}
			}
		})
	}
}

func TestHoltWinters_Forecast_NotInitialized(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)
	hw.Update(10)
	hw.Update(20)

	forecast := hw.Forecast(4)
	if forecast != nil {
		t.Errorf("Forecast should return nil before initialization, got %v", forecast)
	}
}

func TestHoltWinters_Forecast_ZeroSteps(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)
	for i := 0; i < 20; i++ {
		hw.Update(float64(i % 4))
	}

	forecast := hw.Forecast(0)
	if forecast != nil {
		t.Errorf("Forecast(0) should return nil, got %v", forecast)
	}

	forecast = hw.Forecast(-1)
	if forecast != nil {
		t.Errorf("Forecast(-1) should return nil, got %v", forecast)
	}
}

func TestHoltWinters_SinusoidalConvergence(t *testing.T) {
	seasonLen := 24
	hw := NewHoltWinters(0.3, 0.05, 0.3, seasonLen)

	// Generate sinusoidal data: mean=50, amplitude=20, period=24
	genValue := func(i int) float64 {
		return 50.0 + 20.0*math.Sin(2*math.Pi*float64(i)/float64(seasonLen))
	}

	// Feed 5 complete seasons
	totalPoints := 5 * seasonLen
	for i := 0; i < totalPoints; i++ {
		hw.Update(genValue(i))
	}

	if !hw.IsInitialized() {
		t.Fatal("should be initialized after 5 seasons")
	}

	// Verify forecast matches the sinusoidal pattern
	forecast := hw.Forecast(seasonLen)
	if len(forecast) != seasonLen {
		t.Fatalf("Forecast length = %d, want %d", len(forecast), seasonLen)
	}

	maxError := 0.0
	for i, predicted := range forecast {
		expected := genValue(totalPoints + i)
		err := math.Abs(predicted - expected)
		if err > maxError {
			maxError = err
		}
	}

	// After 5 seasons, max forecast error should be reasonable
	if maxError > 10.0 {
		t.Errorf("Max forecast error = %.2f, want < 10.0 for sinusoidal convergence", maxError)
	}
}

func TestHoltWinters_ExpectedRange(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		wantWidth  string // "narrow" or "wide" relative to 95%
	}{
		{
			name:       "90% confidence narrower than 95%",
			confidence: 0.90,
			wantWidth:  "narrow",
		},
		{
			name:       "99% confidence wider than 95%",
			confidence: 0.99,
			wantWidth:  "wide",
		},
	}

	// Build a model with some noise
	seasonLen := 4
	hw := NewHoltWinters(0.3, 0.1, 0.3, seasonLen)
	pattern := []float64{10, 20, 30, 20}
	for cycle := 0; cycle < 5; cycle++ {
		for _, v := range pattern {
			// Add small noise
			noise := float64(cycle%3) - 1.0
			hw.Update(v + noise)
		}
	}

	// Get 95% range as baseline
	lower95, upper95 := hw.ExpectedRange(0.95)
	width95 := upper95 - lower95

	if width95 <= 0 {
		t.Fatalf("95%% range width = %.4f, want > 0", width95)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lower, upper := hw.ExpectedRange(tt.confidence)
			width := upper - lower

			switch tt.wantWidth {
			case "narrow":
				if width >= width95 {
					t.Errorf("Width at %.0f%% = %.4f, should be narrower than 95%% width %.4f",
						tt.confidence*100, width, width95)
				}
			case "wide":
				if width <= width95 {
					t.Errorf("Width at %.0f%% = %.4f, should be wider than 95%% width %.4f",
						tt.confidence*100, width, width95)
				}
			}
		})
	}
}

func TestHoltWinters_ExpectedRange_NotInitialized(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)
	hw.Update(10)

	lower, upper := hw.ExpectedRange(0.95)
	if lower != 0 || upper != 0 {
		t.Errorf("ExpectedRange before init = (%.2f, %.2f), want (0, 0)", lower, upper)
	}
}

func TestHoltWinters_ExpectedRange_ContainsNormalValues(t *testing.T) {
	seasonLen := 4
	pattern := []float64{10, 20, 30, 20}
	hw := NewHoltWinters(0.3, 0.1, 0.3, seasonLen)

	// Train for 5 complete seasons
	for cycle := 0; cycle < 5; cycle++ {
		for _, v := range pattern {
			hw.Update(v)
		}
	}

	// At 99% confidence, the next expected pattern value should be within range
	lower, upper := hw.ExpectedRange(0.99)
	nextExpected := pattern[0] // Next value after 5 full cycles is the first pattern value

	if nextExpected < lower || nextExpected > upper {
		t.Errorf("Expected value %.2f outside 99%% range [%.2f, %.2f]",
			nextExpected, lower, upper)
	}
}

func TestHoltWinters_ResidualStdDev(t *testing.T) {
	tests := []struct {
		name        string
		noise       float64
		wantStdDev  string // "low" or "high"
		maxStdDev   float64
	}{
		{
			name:       "no noise constant pattern",
			noise:      0.0,
			wantStdDev: "low",
			maxStdDev:  1.0,
		},
		{
			name:       "high noise",
			noise:      10.0,
			wantStdDev: "high",
			maxStdDev:  50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seasonLen := 4
			pattern := []float64{10, 20, 30, 20}
			hw := NewHoltWinters(0.3, 0.1, 0.3, seasonLen)

			for cycle := 0; cycle < 10; cycle++ {
				for j, v := range pattern {
					noiseVal := tt.noise * float64((cycle*len(pattern)+j)%5-2) / 2.0
					hw.Update(v + noiseVal)
				}
			}

			sd := hw.ResidualStdDev()
			if sd > tt.maxStdDev {
				t.Errorf("ResidualStdDev = %.4f, want <= %.4f", sd, tt.maxStdDev)
			}
		})
	}
}

func TestHoltWinters_ResidualStdDev_NotInitialized(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)
	hw.Update(10)

	sd := hw.ResidualStdDev()
	if sd != 0 {
		t.Errorf("ResidualStdDev before init = %v, want 0", sd)
	}
}

func TestHoltWinters_ResidualStdDev_JustInitialized(t *testing.T) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 4)
	// Feed exactly one season to initialize
	for i := 0; i < 4; i++ {
		hw.Update(float64(i * 10))
	}

	sd := hw.ResidualStdDev()
	if sd != 0 {
		t.Errorf("ResidualStdDev right at seasonLen = %v, want 0 (not enough post-init samples)", sd)
	}
}

func TestHoltWinters_TrendDetection(t *testing.T) {
	seasonLen := 4
	hw := NewHoltWinters(0.3, 0.3, 0.1, seasonLen)

	// Feed data with upward trend: base increases each cycle
	for cycle := 0; cycle < 10; cycle++ {
		base := float64(cycle * 10)
		hw.Update(base + 10)
		hw.Update(base + 20)
		hw.Update(base + 30)
		hw.Update(base + 20)
	}

	if hw.Trend <= 0 {
		t.Errorf("Trend = %.4f, want > 0 for increasing data", hw.Trend)
	}

	// Forecast should be higher than current level
	forecast := hw.Forecast(4)
	if forecast == nil {
		t.Fatal("Forecast returned nil")
	}

	currentFitted := hw.Fitted()
	avgForecast := 0.0
	for _, v := range forecast {
		avgForecast += v
	}
	avgForecast /= float64(len(forecast))

	if avgForecast <= currentFitted {
		t.Errorf("Average forecast %.2f should be > current fitted %.2f for upward trend",
			avgForecast, currentFitted)
	}
}

func TestConfidenceToZ(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		wantZ      float64
		tolerance  float64
	}{
		{"90% confidence", 0.90, 1.645, 0.02},
		{"95% confidence", 0.95, 1.960, 0.02},
		{"99% confidence", 0.99, 2.576, 0.02},
		{"invalid zero", 0.0, 1.96, 0.01},
		{"invalid one", 1.0, 1.96, 0.01},
		{"invalid negative", -0.5, 1.96, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := confidenceToZ(tt.confidence)
			if math.Abs(z-tt.wantZ) > tt.tolerance {
				t.Errorf("confidenceToZ(%.2f) = %.4f, want %.4f +/- %.4f",
					tt.confidence, z, tt.wantZ, tt.tolerance)
			}
		})
	}
}

func TestInverseNormalCDF(t *testing.T) {
	tests := []struct {
		name      string
		p         float64
		wantZ     float64
		tolerance float64
	}{
		{"p=0.5 -> z=0", 0.5, 0.0, 0.001},
		{"p=0.975 -> z=1.96", 0.975, 1.96, 0.02},
		{"p=0.995 -> z=2.576", 0.995, 2.576, 0.02},
		{"p=0.025 -> z=-1.96", 0.025, -1.96, 0.02},
		{"p=0 boundary", 0.0, 0.0, 0.001},
		{"p=1 boundary", 1.0, 0.0, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := inverseNormalCDF(tt.p)
			if math.Abs(z-tt.wantZ) > tt.tolerance {
				t.Errorf("inverseNormalCDF(%.4f) = %.4f, want %.4f +/- %.4f",
					tt.p, z, tt.wantZ, tt.tolerance)
			}
		})
	}
}

func TestHoltWinters_DualSeasonLength(t *testing.T) {
	tests := []struct {
		name      string
		seasonLen int
	}{
		{"daily 24-point season", 24},
		{"weekly 168-point season", 168},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewHoltWinters(0.3, 0.1, 0.3, tt.seasonLen)

			// Feed 3 complete seasons of sinusoidal data
			for i := 0; i < 3*tt.seasonLen; i++ {
				v := 50.0 + 20.0*math.Sin(2*math.Pi*float64(i)/float64(tt.seasonLen))
				hw.Update(v)
			}

			if !hw.IsInitialized() {
				t.Fatal("should be initialized after 3 seasons")
			}
			if hw.SeasonLen != tt.seasonLen {
				t.Errorf("SeasonLen = %d, want %d", hw.SeasonLen, tt.seasonLen)
			}

			// Verify forecast returns correct number of steps
			forecast := hw.Forecast(tt.seasonLen)
			if len(forecast) != tt.seasonLen {
				t.Errorf("Forecast length = %d, want %d", len(forecast), tt.seasonLen)
			}

			// Verify expected range is valid
			lower, upper := hw.ExpectedRange(0.95)
			if lower >= upper {
				t.Errorf("ExpectedRange lower %.2f >= upper %.2f", lower, upper)
			}
		})
	}
}

func BenchmarkHoltWinters_Forecast(b *testing.B) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 24)
	for i := 0; i < 120; i++ {
		hw.Update(50.0 + 20.0*math.Sin(2*math.Pi*float64(i)/24.0))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hw.Forecast(24)
	}
}

func BenchmarkHoltWinters_ExpectedRange(b *testing.B) {
	hw := NewHoltWinters(0.3, 0.1, 0.3, 24)
	for i := 0; i < 120; i++ {
		hw.Update(50.0 + 20.0*math.Sin(2*math.Pi*float64(i)/24.0))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hw.ExpectedRange(0.95)
	}
}
