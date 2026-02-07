package anomaly

import "math"

// Severity levels for detected anomalies.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// ZScoreResult contains the result of a Z-score check.
type ZScoreResult struct {
	IsAnomaly bool
	ZScore    float64
	Severity  string
}

// ZScoreCheck evaluates whether a value is anomalous given a baseline mean and standard deviation.
// threshold is the minimum |z-score| to flag as anomalous (e.g., 3.0).
// Severity mapping:
//   - warning: |z| >= threshold and |z| < threshold+1
//   - critical: |z| >= threshold+1
func ZScoreCheck(value, mean, stdDev, threshold float64) ZScoreResult {
	if stdDev <= 0 {
		return ZScoreResult{}
	}
	z := (value - mean) / stdDev
	absZ := math.Abs(z)

	if absZ < threshold {
		return ZScoreResult{ZScore: z}
	}

	severity := SeverityWarning
	if absZ >= threshold+1 {
		severity = SeverityCritical
	}

	return ZScoreResult{
		IsAnomaly: true,
		ZScore:    z,
		Severity:  severity,
	}
}
