// Package analytics provides public SDK types for the SubNetree analytics system.
// This package is Apache 2.0 licensed, part of the public plugin SDK.
package analytics

import "time"

// Anomaly represents a detected anomaly on a device metric.
type Anomaly struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	MetricName  string     `json:"metric_name"`
	Severity    string     `json:"severity"`     // "info", "warning", "critical"
	Type        string     `json:"type"`         // "zscore", "cusum", "trend"
	Value       float64    `json:"value"`        // Observed value
	Expected    float64    `json:"expected"`     // Baseline expected value
	Deviation   float64    `json:"deviation"`    // How far from baseline (sigma)
	DetectedAt  time.Time  `json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Description string     `json:"description"`
}

// Forecast represents a capacity forecast for a device metric.
type Forecast struct {
	DeviceID        string         `json:"device_id"`
	MetricName      string         `json:"metric_name"`
	CurrentValue    float64        `json:"current_value"`
	PredictedValue  float64        `json:"predicted_value"`
	TimeToThreshold *time.Duration `json:"time_to_threshold,omitempty"`
	Threshold       float64        `json:"threshold"`
	Confidence      float64        `json:"confidence"` // 0.0-1.0
	Slope           float64        `json:"slope"`      // Rate of change per hour
	GeneratedAt     time.Time      `json:"generated_at"`
}

// Baseline represents a learned baseline for a device metric.
type Baseline struct {
	DeviceID   string    `json:"device_id"`
	MetricName string    `json:"metric_name"`
	Algorithm  string    `json:"algorithm"` // "ewma", "holt_winters"
	Mean       float64   `json:"mean"`
	StdDev     float64   `json:"std_dev"`
	Samples    int       `json:"samples"`
	Stable     bool      `json:"stable"` // true after learning period
	UpdatedAt  time.Time `json:"updated_at"`
}

// AlertGroup represents a group of correlated alerts.
type AlertGroup struct {
	ID          string    `json:"id"`
	RootCause   string    `json:"root_cause,omitempty"` // Device ID identified as root
	DeviceIDs   []string  `json:"device_ids"`
	AlertCount  int       `json:"alert_count"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
}

// NLQueryRequest is the request body for POST /analytics/query.
type NLQueryRequest struct {
	Query string `json:"query"`
}

// NLQueryResponse is the response for POST /analytics/query.
type NLQueryResponse struct {
	Query      string `json:"query"`
	Answer     string `json:"answer"`
	Structured any    `json:"structured,omitempty"`
	Model      string `json:"model,omitempty"`
}

// MetricPoint is a single time-series data point for analytics processing.
type MetricPoint struct {
	DeviceID   string            `json:"device_id"`
	MetricName string            `json:"metric_name"`
	Value      float64           `json:"value"`
	Timestamp  time.Time         `json:"timestamp"`
	Tags       map[string]string `json:"tags,omitempty"`
}
