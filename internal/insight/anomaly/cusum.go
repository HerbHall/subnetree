package anomaly

import "math"

// CUSUMResult contains the result of a CUSUM check.
type CUSUMResult struct {
	IsChangePoint bool
	Direction     string  // "up" or "down"
	CUSUMHigh     float64 // Upper cumulative sum
	CUSUMLow      float64 // Lower cumulative sum
}

// CUSUM tracks cumulative sums for change-point detection.
type CUSUM struct {
	Drift     float64 // Allowable drift (slack parameter k)
	Threshold float64 // Decision threshold (h)
	High      float64 // Upper cumulative sum S+
	Low       float64 // Lower cumulative sum S-
}

// NewCUSUM creates a new CUSUM detector.
func NewCUSUM(drift, threshold float64) *CUSUM {
	return &CUSUM{
		Drift:     drift,
		Threshold: threshold,
	}
}

// Update processes a new normalized value (value - mean) / stdDev and checks for a change point.
// Pass (value - baseline_mean) / baseline_stdDev as the normalized value.
func (c *CUSUM) Update(normalized float64) CUSUMResult {
	c.High = math.Max(0, c.High+normalized-c.Drift)
	c.Low = math.Max(0, c.Low-normalized-c.Drift)

	result := CUSUMResult{
		CUSUMHigh: c.High,
		CUSUMLow:  c.Low,
	}

	if c.High > c.Threshold {
		result.IsChangePoint = true
		result.Direction = "up"
		c.High = 0 // Reset after detection
	}
	if c.Low > c.Threshold {
		result.IsChangePoint = true
		result.Direction = "down"
		c.Low = 0 // Reset after detection
	}

	return result
}

// Reset resets the CUSUM accumulators.
func (c *CUSUM) Reset() {
	c.High = 0
	c.Low = 0
}
