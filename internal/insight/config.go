package insight

import "time"

// InsightConfig holds configuration for the Insight analytics plugin.
type InsightConfig struct {
	EWMAAlpha           float64       `mapstructure:"ewma_alpha"`
	LearningPeriod      time.Duration `mapstructure:"learning_period"`
	MinSamplesStable    int           `mapstructure:"min_samples_stable"`
	ZScoreThreshold     float64       `mapstructure:"zscore_threshold"`
	CUSUMDrift          float64       `mapstructure:"cusum_drift"`
	CUSUMThreshold      float64       `mapstructure:"cusum_threshold"`
	ForecastWindow      time.Duration `mapstructure:"forecast_window"`
	AnomalyRetention    time.Duration `mapstructure:"anomaly_retention"`
	MaintenanceInterval time.Duration `mapstructure:"maintenance_interval"`
}

// DefaultConfig returns sensible defaults for the Insight module.
func DefaultConfig() InsightConfig {
	return InsightConfig{
		EWMAAlpha:           0.1,
		LearningPeriod:      7 * 24 * time.Hour,
		MinSamplesStable:    168,
		ZScoreThreshold:     3.0,
		CUSUMDrift:          0.5,
		CUSUMThreshold:      5.0,
		ForecastWindow:      7 * 24 * time.Hour,
		AnomalyRetention:    30 * 24 * time.Hour,
		MaintenanceInterval: 1 * time.Hour,
	}
}
