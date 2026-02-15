package insight

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"go.uber.org/zap"
)

// Optimizer analyzes Pulse metrics and generates optimization recommendations.
type Optimizer struct {
	store  *InsightStore
	logger *zap.Logger
}

// NewOptimizer creates a new optimizer.
func NewOptimizer(store *InsightStore, logger *zap.Logger) *Optimizer {
	return &Optimizer{store: store, logger: logger}
}

// thresholds for resource utilization recommendations.
var thresholds = map[string]struct {
	warning  float64
	critical float64
}{
	"cpu_percent":    {warning: 80, critical: 95},
	"memory_percent": {warning: 85, critical: 95},
	"disk_percent":   {warning: 90, critical: 95},
}

// GenerateRecommendations analyzes recent metrics and returns recommendations.
func (o *Optimizer) GenerateRecommendations(ctx context.Context) ([]analytics.Recommendation, error) {
	if o.store == nil {
		return []analytics.Recommendation{}, nil
	}

	// Get the latest metric value per device+metric combination
	since := time.Now().Add(-1 * time.Hour)
	var recs []analytics.Recommendation

	for metricName, thresh := range thresholds {
		latest, err := o.store.GetLatestMetricPerDevice(ctx, metricName, since)
		if err != nil {
			o.logger.Debug("failed to query metrics for recommendations",
				zap.String("metric", metricName),
				zap.Error(err),
			)
			continue
		}

		for _, point := range latest {
			if point.Value >= thresh.critical {
				recs = append(recs, analytics.Recommendation{
					ID:           fmt.Sprintf("rec:%s:%s:%d", point.DeviceID, metricName, time.Now().UnixNano()),
					DeviceID:     point.DeviceID,
					Type:         metricType(metricName),
					Severity:     "critical",
					Title:        fmt.Sprintf("Critical %s usage on device", metricType(metricName)),
					Description:  fmt.Sprintf("%s is at %.1f%%, exceeding the critical threshold of %.0f%%. Immediate attention recommended.", metricName, point.Value, thresh.critical),
					Metric:       metricName,
					CurrentValue: point.Value,
					Threshold:    thresh.critical,
					GeneratedAt:  time.Now(),
				})
			} else if point.Value >= thresh.warning {
				recs = append(recs, analytics.Recommendation{
					ID:           fmt.Sprintf("rec:%s:%s:%d", point.DeviceID, metricName, time.Now().UnixNano()),
					DeviceID:     point.DeviceID,
					Type:         metricType(metricName),
					Severity:     "warning",
					Title:        fmt.Sprintf("High %s usage on device", metricType(metricName)),
					Description:  fmt.Sprintf("%s is at %.1f%%, exceeding the warning threshold of %.0f%%. Consider investigating.", metricName, point.Value, thresh.warning),
					Metric:       metricName,
					CurrentValue: point.Value,
					Threshold:    thresh.warning,
					GeneratedAt:  time.Now(),
				})
			}
		}
	}

	return recs, nil
}

func metricType(metricName string) string {
	switch metricName {
	case "cpu_percent":
		return "cpu"
	case "memory_percent":
		return "memory"
	case "disk_percent":
		return "disk"
	default:
		return metricName
	}
}
