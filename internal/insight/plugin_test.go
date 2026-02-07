package insight

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/config"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"github.com/HerbHall/subnetree/pkg/roles"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInit_WithConfig(t *testing.T) {
	v := viper.New()
	v.Set("ewma_alpha", 0.2)
	v.Set("zscore_threshold", 2.5)
	v.Set("learning_period", "48h")

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	err = m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
		Store:  db,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if m.cfg.EWMAAlpha != 0.2 {
		t.Errorf("cfg.EWMAAlpha = %f, want 0.2", m.cfg.EWMAAlpha)
	}
	if m.cfg.ZScoreThreshold != 2.5 {
		t.Errorf("cfg.ZScoreThreshold = %f, want 2.5", m.cfg.ZScoreThreshold)
	}
	if m.cfg.LearningPeriod != 48*time.Hour {
		t.Errorf("cfg.LearningPeriod = %v, want 48h", m.cfg.LearningPeriod)
	}
}

func TestInit_NilConfig(t *testing.T) {
	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Init() with nil config error = %v", err)
	}

	// Verify defaults are applied.
	defaults := DefaultConfig()
	if m.cfg.EWMAAlpha != defaults.EWMAAlpha {
		t.Errorf("cfg.EWMAAlpha = %f, want default %f", m.cfg.EWMAAlpha, defaults.EWMAAlpha)
	}
	if m.cfg.ZScoreThreshold != defaults.ZScoreThreshold {
		t.Errorf("cfg.ZScoreThreshold = %f, want default %f", m.cfg.ZScoreThreshold, defaults.ZScoreThreshold)
	}
}

func TestInfo_HasCorrectRoles(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "insight" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "insight")
	}
	if info.Required {
		t.Error("Info().Required = true, want false")
	}

	found := false
	for _, r := range info.Roles {
		if r == roles.RoleAnalytics {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Info().Roles = %v, want to contain %q", info.Roles, roles.RoleAnalytics)
	}
}

func TestHealth_ReportsStatus(t *testing.T) {
	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())

	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
	if _, ok := status.Details["baselines_tracked"]; !ok {
		t.Error("Health().Details missing key \"baselines_tracked\"")
	}
	if _, ok := status.Details["llm_available"]; !ok {
		t.Error("Health().Details missing key \"llm_available\"")
	}
}

func TestSubscriptions_ReturnsTopics(t *testing.T) {
	m := New()

	subs := m.Subscriptions()
	if len(subs) != 4 {
		t.Fatalf("Subscriptions() returned %d, want 4", len(subs))
	}

	expected := map[string]bool{
		TopicMetricsCollected: false,
		TopicAlertTriggered:   false,
		TopicAlertResolved:    false,
		TopicDeviceDiscovered: false,
	}
	for _, s := range subs {
		if _, ok := expected[s.Topic]; !ok {
			t.Errorf("unexpected subscription topic: %q", s.Topic)
		}
		expected[s.Topic] = true
		if s.Handler == nil {
			t.Errorf("subscription for %q has nil handler", s.Topic)
		}
	}
	for topic, seen := range expected {
		if !seen {
			t.Errorf("missing subscription for topic %q", topic)
		}
	}
}

func TestAnalyticsProvider_EmptyResults(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	err = m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Store:  db,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx := context.Background()

	anomalies, err := m.Anomalies(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Anomalies() error = %v", err)
	}
	if anomalies != nil {
		t.Errorf("Anomalies() = %v, want nil (empty)", anomalies)
	}

	baselines, err := m.Baselines(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Baselines() error = %v", err)
	}
	if baselines != nil {
		t.Errorf("Baselines() = %v, want nil (empty)", baselines)
	}

	forecasts, err := m.Forecasts(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Forecasts() error = %v", err)
	}
	if forecasts != nil {
		t.Errorf("Forecasts() = %v, want nil (empty)", forecasts)
	}
}
