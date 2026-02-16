package recon

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// HealthScoreFactor represents one weighted component of the overall health score.
type HealthScoreFactor struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

// HealthScoreResponse is the API response for GET /metrics/health-score.
type HealthScoreResponse struct {
	Score   int                 `json:"score"`
	Grade   string              `json:"grade"`
	Factors []HealthScoreFactor `json:"factors"`
}

// computeHealthScore calculates a 0-100 health score from recent scan metrics.
// Factors and weights:
//   - Scan success rate (25%): completed vs total scans
//   - Duration stability (20%): coefficient of variation of durations
//   - Device count stability (20%): coefficient of variation of device counts
//   - Average ping RTT proxy (15%): duration_ms / hosts_scanned
//   - DNS lookup success (10%): default 95% (not tracked per-scan)
//   - New device rate (10%): devices_created per week average
func computeHealthScore(metrics []models.ScanMetrics, scans []models.ScanResult) HealthScoreResponse {
	factors := make([]HealthScoreFactor, 0, 6)

	// Factor 1: Scan success rate (25%)
	successScore, successDetail := computeSuccessRate(scans)
	factors = append(factors, HealthScoreFactor{
		Name:   "scan_success_rate",
		Score:  successScore,
		Weight: 0.25,
		Detail: successDetail,
	})

	// Factor 2: Duration stability (20%)
	durationScore, durationDetail := computeDurationStability(metrics)
	factors = append(factors, HealthScoreFactor{
		Name:   "duration_stability",
		Score:  durationScore,
		Weight: 0.20,
		Detail: durationDetail,
	})

	// Factor 3: Device count stability (20%)
	deviceScore, deviceDetail := computeDeviceCountStability(metrics)
	factors = append(factors, HealthScoreFactor{
		Name:   "device_count_stability",
		Score:  deviceScore,
		Weight: 0.20,
		Detail: deviceDetail,
	})

	// Factor 4: Average ping RTT proxy (15%)
	rttScore, rttDetail := computeAvgRTTProxy(metrics)
	factors = append(factors, HealthScoreFactor{
		Name:   "avg_ping_rtt",
		Score:  rttScore,
		Weight: 0.15,
		Detail: rttDetail,
	})

	// Factor 5: DNS lookup success (10%) - default 95%
	factors = append(factors, HealthScoreFactor{
		Name:   "dns_lookup_success",
		Score:  95.0,
		Weight: 0.10,
		Detail: "default 95% (not tracked per-scan)",
	})

	// Factor 6: New device rate (10%)
	newDeviceScore, newDeviceDetail := computeNewDeviceRate(metrics)
	factors = append(factors, HealthScoreFactor{
		Name:   "new_device_rate",
		Score:  newDeviceScore,
		Weight: 0.10,
		Detail: newDeviceDetail,
	})

	// Compute weighted score.
	var totalScore float64
	for i := range factors {
		totalScore += factors[i].Score * factors[i].Weight
	}

	score := int(math.Round(totalScore))
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	grade := "red"
	if score >= 80 {
		grade = "green"
	} else if score >= 60 {
		grade = "yellow"
	}

	return HealthScoreResponse{
		Score:   score,
		Grade:   grade,
		Factors: factors,
	}
}

// computeSuccessRate returns a 0-100 score for scan success rate.
func computeSuccessRate(scans []models.ScanResult) (float64, string) {
	if len(scans) == 0 {
		return 100.0, "no scans to evaluate"
	}

	completed := 0
	for i := range scans {
		if scans[i].Status == "completed" {
			completed++
		}
	}

	rate := float64(completed) / float64(len(scans)) * 100.0
	detail := strconv.Itoa(completed) + "/" + strconv.Itoa(len(scans)) + " scans completed"
	return rate, detail
}

// computeDurationStability returns a 0-100 score based on scan duration consistency.
// Lower coefficient of variation = higher score.
func computeDurationStability(metrics []models.ScanMetrics) (float64, string) {
	if len(metrics) < 2 {
		return 100.0, "insufficient data for stability analysis"
	}

	durations := make([]float64, 0, len(metrics))
	for i := range metrics {
		if metrics[i].DurationMs > 0 {
			durations = append(durations, float64(metrics[i].DurationMs))
		}
	}

	if len(durations) < 2 {
		return 100.0, "insufficient non-zero durations"
	}

	cv := coefficientOfVariation(durations)
	// CV < 0.1 = perfect (100), CV > 1.0 = poor (0), linear between.
	score := 100.0 * (1.0 - clamp(cv/1.0, 0, 1))
	detail := fmt.Sprintf("%.1f%% coefficient of variation", cv*100)
	return score, detail
}

// computeDeviceCountStability returns a 0-100 score based on device count consistency.
func computeDeviceCountStability(metrics []models.ScanMetrics) (float64, string) {
	if len(metrics) < 2 {
		return 100.0, "insufficient data for stability analysis"
	}

	counts := make([]float64, 0, len(metrics))
	for i := range metrics {
		counts = append(counts, float64(metrics[i].HostsAlive))
	}

	cv := coefficientOfVariation(counts)
	score := 100.0 * (1.0 - clamp(cv/1.0, 0, 1))
	detail := fmt.Sprintf("%.1f%% coefficient of variation", cv*100)
	return score, detail
}

// computeAvgRTTProxy returns a 0-100 score approximating network latency.
// Uses ping_phase_ms / hosts_scanned as a proxy for average per-host time.
func computeAvgRTTProxy(metrics []models.ScanMetrics) (float64, string) {
	if len(metrics) == 0 {
		return 100.0, "no metrics available"
	}

	var totalRTT float64
	var count int
	for i := range metrics {
		if metrics[i].HostsScanned > 0 {
			rtt := float64(metrics[i].PingPhaseMs) / float64(metrics[i].HostsScanned)
			totalRTT += rtt
			count++
		}
	}

	if count == 0 {
		return 100.0, "no scans with hosts scanned"
	}

	avgRTT := totalRTT / float64(count)
	// < 5ms = perfect (100), > 200ms = poor (0), logarithmic scale.
	var score float64
	switch {
	case avgRTT <= 5:
		score = 100.0
	case avgRTT >= 200:
		score = 0.0
	default:
		// Log scale: score = 100 * (1 - log(rtt/5) / log(200/5))
		score = 100.0 * (1.0 - math.Log(avgRTT/5.0)/math.Log(200.0/5.0))
	}

	detail := fmt.Sprintf("%.1f ms avg per-host ping time", avgRTT)
	return clamp(score, 0, 100), detail
}

// computeNewDeviceRate returns a 0-100 score for the rate of new device discovery.
// A moderate new-device rate indicates healthy network monitoring.
func computeNewDeviceRate(metrics []models.ScanMetrics) (float64, string) {
	if len(metrics) == 0 {
		return 80.0, "no metrics available"
	}

	var totalNewDevices int
	for i := range metrics {
		totalNewDevices += metrics[i].DevicesCreated
	}

	// Calculate the time span covered.
	if len(metrics) < 2 {
		if totalNewDevices > 0 {
			return 90.0, strconv.Itoa(totalNewDevices) + " new devices discovered"
		}
		return 80.0, "single scan, no trend available"
	}

	earliest, _ := time.Parse(time.RFC3339, metrics[0].CreatedAt)
	latest, _ := time.Parse(time.RFC3339, metrics[len(metrics)-1].CreatedAt)
	spanDays := latest.Sub(earliest).Hours() / 24.0
	if spanDays < 1 {
		spanDays = 1
	}

	weeklyRate := float64(totalNewDevices) / spanDays * 7.0

	// 0-2 new devices/week = great (100), 3-10 = good (80-90), >10 = concerning (60-80)
	var score float64
	switch {
	case weeklyRate <= 2:
		score = 100.0
	case weeklyRate <= 10:
		score = 90.0 - (weeklyRate-2.0)*1.25
	default:
		score = 80.0 - clamp((weeklyRate-10.0)*2.0, 0, 40)
	}

	detail := fmt.Sprintf("%.1f new devices/week", weeklyRate)
	return clamp(score, 0, 100), detail
}

// coefficientOfVariation returns the CV of a slice of float64 values.
func coefficientOfVariation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	if mean == 0 {
		return 0
	}

	var sqDiffSum float64
	for _, v := range values {
		diff := v - mean
		sqDiffSum += diff * diff
	}
	stdDev := math.Sqrt(sqDiffSum / float64(len(values)))

	return stdDev / mean
}

// clamp restricts a value to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// getHealthScore fetches recent metrics and scans, then computes the health score.
func (m *Module) getHealthScore(ctx context.Context) (HealthScoreResponse, error) {
	since := time.Now().Add(-30 * 24 * time.Hour) // Last 30 days
	metrics, err := m.store.GetRawMetricsSince(ctx, since)
	if err != nil {
		return HealthScoreResponse{}, err
	}

	scans, err := m.store.ListScans(ctx, 100, 0)
	if err != nil {
		return HealthScoreResponse{}, err
	}

	return computeHealthScore(metrics, scans), nil
}
