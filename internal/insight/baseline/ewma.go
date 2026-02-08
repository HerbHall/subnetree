package baseline

import "math"

// EWMA tracks an exponentially weighted moving average with online variance estimation.
type EWMA struct {
	Alpha   float64 // Smoothing factor (0 < alpha <= 1)
	Mean    float64 // Current smoothed mean
	Var     float64 // Online variance estimate (Welford's method)
	Samples int     // Number of samples processed
}

// NewEWMA creates a new EWMA tracker with the given smoothing factor.
func NewEWMA(alpha float64) *EWMA {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.1
	}
	return &EWMA{Alpha: alpha}
}

// Update processes a new value and updates the mean and variance.
func (e *EWMA) Update(value float64) {
	e.Samples++
	if e.Samples == 1 {
		e.Mean = value
		e.Var = 0
		return
	}
	diff := value - e.Mean
	e.Mean += e.Alpha * diff
	// Welford-style online variance using EWMA weighting
	e.Var = (1 - e.Alpha) * (e.Var + e.Alpha*diff*diff)
}

// StdDev returns the current standard deviation.
func (e *EWMA) StdDev() float64 {
	if e.Samples < 2 {
		return 0
	}
	return math.Sqrt(e.Var)
}
