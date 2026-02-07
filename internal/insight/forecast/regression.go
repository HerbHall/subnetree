package forecast

import "time"

// RegressionResult contains the output of a linear regression.
type RegressionResult struct {
	Slope       float64        // Rate of change per hour
	Intercept   float64        // Y-intercept (value at t=0)
	RSquared    float64        // Coefficient of determination (0-1)
	Predicted   float64        // Predicted current value from the model
	TimeToLimit *time.Duration // Time until threshold is reached (nil if not approaching)
}

// LinearRegression performs least-squares regression on time-series data.
// times should be in hours relative to some epoch. values are the metric values.
// threshold is the capacity limit to forecast against.
// Returns nil if fewer than 2 points are provided.
func LinearRegression(times, values []float64, threshold float64) *RegressionResult {
	n := len(times)
	if n < 2 || len(values) != n {
		return nil
	}

	// Calculate means
	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += times[i]
		sumY += values[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Calculate slope and intercept using least squares
	var ssXY, ssXX, ssYY float64
	for i := 0; i < n; i++ {
		dx := times[i] - meanX
		dy := values[i] - meanY
		ssXY += dx * dy
		ssXX += dx * dx
		ssYY += dy * dy
	}

	if ssXX == 0 {
		return &RegressionResult{
			Slope:     0,
			Intercept: meanY,
			Predicted: meanY,
		}
	}

	slope := ssXY / ssXX
	intercept := meanY - slope*meanX

	// R-squared
	var rSquared float64
	if ssYY > 0 {
		rSquared = (ssXY * ssXY) / (ssXX * ssYY)
	}

	// Predicted value at the last time point
	predicted := slope*times[n-1] + intercept

	result := &RegressionResult{
		Slope:     slope,
		Intercept: intercept,
		RSquared:  rSquared,
		Predicted: predicted,
	}

	// Time to threshold: solve slope*t + intercept = threshold
	// Only if slope is positive and threshold is above current, or slope is negative and threshold is below current
	if slope > 0 && predicted < threshold {
		hoursToLimit := (threshold - predicted) / slope
		d := time.Duration(hoursToLimit * float64(time.Hour))
		result.TimeToLimit = &d
	} else if slope < 0 && predicted > threshold {
		hoursToLimit := (threshold - predicted) / slope // negative/negative = positive
		d := time.Duration(hoursToLimit * float64(time.Hour))
		result.TimeToLimit = &d
	}

	return result
}

// TimeToHours converts time.Time values to hours relative to the first point.
func TimeToHours(timestamps []time.Time) []float64 {
	if len(timestamps) == 0 {
		return nil
	}
	hours := make([]float64, len(timestamps))
	base := timestamps[0]
	for i, t := range timestamps {
		hours[i] = t.Sub(base).Hours()
	}
	return hours
}
