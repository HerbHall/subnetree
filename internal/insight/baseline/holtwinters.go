package baseline

import "math"

// HoltWinters implements triple exponential smoothing with additive seasonality.
// It tracks level, trend, and seasonal components along with an online residual
// variance estimate for confidence interval computation.
type HoltWinters struct {
	Alpha       float64   // Level smoothing (0-1)
	Beta        float64   // Trend smoothing (0-1)
	Gamma       float64   // Seasonal smoothing (0-1)
	SeasonLen   int       // Number of points in one season
	Level       float64   // Current level
	Trend       float64   // Current trend
	Seasonal    []float64 // Seasonal components
	Samples     int       // Total samples processed
	ResidualVar float64   // EWMA of squared residuals (online variance)
	initialized bool
}

// NewHoltWinters creates a new Holt-Winters tracker.
// seasonLen is the number of data points per season (e.g., 24 for hourly data
// with daily seasonality, or 168 for weekly seasonality with hourly samples).
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

// Update processes a new value and updates level, trend, seasonal components,
// and the residual variance estimate.
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

	// Compute one-step-ahead fitted value before updating components
	fitted := hw.Level + hw.Trend + hw.Seasonal[idx]
	residual := value - fitted

	prevLevel := hw.Level
	// Level update
	hw.Level = hw.Alpha*(value-hw.Seasonal[idx]) + (1-hw.Alpha)*(prevLevel+hw.Trend)
	// Trend update
	hw.Trend = hw.Beta*(hw.Level-prevLevel) + (1-hw.Beta)*hw.Trend
	// Seasonal update
	hw.Seasonal[idx] = hw.Gamma*(value-hw.Level) + (1-hw.Gamma)*hw.Seasonal[idx]

	// Online residual variance (EWMA of squared residuals)
	hw.ResidualVar = (1-hw.Alpha)*hw.ResidualVar + hw.Alpha*residual*residual
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

// Forecast returns forecasted values for 1..steps into the future.
// Returns nil if the model is not yet initialized.
func (hw *HoltWinters) Forecast(steps int) []float64 {
	if !hw.initialized || steps <= 0 {
		return nil
	}
	result := make([]float64, steps)
	for i := range steps {
		result[i] = hw.Predict(i + 1)
	}
	return result
}

// Fitted returns the expected value for the most recent data point.
func (hw *HoltWinters) Fitted() float64 {
	if !hw.initialized {
		return 0
	}
	idx := (hw.Samples - 1) % hw.SeasonLen
	return hw.Level + hw.Seasonal[idx]
}

// ResidualStdDev returns the standard deviation of the model's residuals.
// This measures how well the model fits recent data.
func (hw *HoltWinters) ResidualStdDev() float64 {
	if !hw.initialized || hw.Samples <= hw.SeasonLen {
		return 0
	}
	return math.Sqrt(hw.ResidualVar)
}

// ExpectedRange returns the lower and upper bounds of the expected range
// for the next data point at the given confidence level (0-1).
// Uses a Gaussian approximation: fitted +/- z * residualStdDev.
// Common confidence values: 0.90 -> z=1.645, 0.95 -> z=1.96, 0.99 -> z=2.576.
func (hw *HoltWinters) ExpectedRange(confidence float64) (lower, upper float64) {
	if !hw.initialized {
		return 0, 0
	}
	fitted := hw.Predict(1)
	sd := hw.ResidualStdDev()
	if sd <= 0 {
		return fitted, fitted
	}

	z := confidenceToZ(confidence)
	margin := z * sd
	return fitted - margin, fitted + margin
}

// IsInitialized returns true if enough data has been seen to start forecasting.
func (hw *HoltWinters) IsInitialized() bool {
	return hw.initialized
}

// confidenceToZ converts a confidence level (0-1) to the corresponding
// z-score for a two-tailed Gaussian distribution.
// Uses rational approximation of the inverse normal CDF (Abramowitz & Stegun 26.2.23).
func confidenceToZ(confidence float64) float64 {
	if confidence <= 0 || confidence >= 1 {
		return 1.96 // default to 95%
	}
	// Two-tailed: p = (1 + confidence) / 2
	p := (1 + confidence) / 2
	return inverseNormalCDF(p)
}

// inverseNormalCDF approximates the inverse of the standard normal CDF.
// Uses the rational approximation from Abramowitz & Stegun (26.2.23).
func inverseNormalCDF(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	if p < 0.5 {
		return -inverseNormalCDF(1 - p)
	}

	// Rational approximation for 0.5 <= p < 1
	t := math.Sqrt(-2 * math.Log(1-p))

	// Coefficients from Abramowitz & Stegun
	const (
		c0 = 2.515517
		c1 = 0.802853
		c2 = 0.010328
		d1 = 1.432788
		d2 = 0.189269
		d3 = 0.001308
	)

	return t - (c0+c1*t+c2*t*t)/(1+d1*t+d2*t*t+d3*t*t*t)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
