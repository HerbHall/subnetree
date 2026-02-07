package baseline

import "math"

// HoltWinters implements triple exponential smoothing with additive seasonality.
type HoltWinters struct {
	Alpha       float64   // Level smoothing (0-1)
	Beta        float64   // Trend smoothing (0-1)
	Gamma       float64   // Seasonal smoothing (0-1)
	SeasonLen   int       // Number of points in one season
	Level       float64   // Current level
	Trend       float64   // Current trend
	Seasonal    []float64 // Seasonal components
	Samples     int       // Total samples processed
	initialized bool
}

// NewHoltWinters creates a new Holt-Winters tracker.
// seasonLen is the number of data points per season (e.g., 24 for hourly data with daily seasonality).
func NewHoltWinters(alpha, beta, gamma float64, seasonLen int) *HoltWinters {
	if seasonLen < 2 {
		seasonLen = 24
	}
	return &HoltWinters{
		Alpha:     clamp(alpha, 0, 1),
		Beta:      clamp(beta, 0, 1),
		Gamma:     clamp(gamma, 0, 1),
		SeasonLen: seasonLen,
		Seasonal:  make([]float64, seasonLen),
	}
}

// Update processes a new value and updates level, trend, and seasonal components.
func (hw *HoltWinters) Update(value float64) {
	hw.Samples++
	idx := (hw.Samples - 1) % hw.SeasonLen

	if !hw.initialized {
		// Accumulate initial season
		hw.Seasonal[idx] = value
		if hw.Samples == hw.SeasonLen {
			hw.initialize()
		}
		return
	}

	prevLevel := hw.Level
	// Level update
	hw.Level = hw.Alpha*(value-hw.Seasonal[idx]) + (1-hw.Alpha)*(prevLevel+hw.Trend)
	// Trend update
	hw.Trend = hw.Beta*(hw.Level-prevLevel) + (1-hw.Beta)*hw.Trend
	// Seasonal update
	hw.Seasonal[idx] = hw.Gamma*(value-hw.Level) + (1-hw.Gamma)*hw.Seasonal[idx]
}

// initialize sets initial level, trend, and seasonal components from the first season.
func (hw *HoltWinters) initialize() {
	hw.initialized = true
	// Level = average of first season
	sum := 0.0
	for _, v := range hw.Seasonal {
		sum += v
	}
	hw.Level = sum / float64(hw.SeasonLen)

	// Trend = 0 (single season, no trend yet)
	hw.Trend = 0

	// Seasonal = deviation from mean
	for i := range hw.Seasonal {
		hw.Seasonal[i] -= hw.Level
	}
}

// Predict returns the forecasted value for stepsAhead into the future.
func (hw *HoltWinters) Predict(stepsAhead int) float64 {
	if !hw.initialized {
		return 0
	}
	idx := (hw.Samples + stepsAhead - 1) % hw.SeasonLen
	return hw.Level + float64(stepsAhead)*hw.Trend + hw.Seasonal[idx]
}

// Fitted returns the expected value for the most recent data point.
func (hw *HoltWinters) Fitted() float64 {
	if !hw.initialized {
		return 0
	}
	idx := (hw.Samples - 1) % hw.SeasonLen
	return hw.Level + hw.Seasonal[idx]
}

// IsInitialized returns true if enough data has been seen to start forecasting.
func (hw *HoltWinters) IsInitialized() bool {
	return hw.initialized
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
