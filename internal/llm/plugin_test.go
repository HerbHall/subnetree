package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HerbHall/subnetree/internal/config"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInit_WithConfig(t *testing.T) {
	srv := mockHeartbeat(t)

	v := viper.New()
	v.Set("provider", "ollama")
	v.Set("ollama.url", srv.URL)
	v.Set("ollama.model", "test-model")
	v.Set("ollama.timeout", "30s")

	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if m.provider == nil {
		t.Fatal("provider is nil after Init")
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
	if m.provider == nil {
		t.Fatal("provider is nil after Init with nil config")
	}
}

func TestStart_HeartbeatFails(t *testing.T) {
	// Point at a closed server -- Start should succeed with a warning, not error.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	v := viper.New()
	v.Set("provider", "ollama")
	v.Set("ollama.url", srv.URL)
	v.Set("ollama.timeout", "1s")

	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() should succeed even when Ollama is unreachable, got error = %v", err)
	}
}

func TestHealth_Healthy(t *testing.T) {
	srv := mockHeartbeat(t)

	v := viper.New()
	v.Set("provider", "ollama")
	v.Set("ollama.url", srv.URL)
	v.Set("ollama.timeout", "5s")

	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	v := viper.New()
	v.Set("provider", "ollama")
	v.Set("ollama.url", srv.URL)
	v.Set("ollama.timeout", "1s")

	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "unhealthy")
	}
	if status.Message == "" {
		t.Error("Health().Message should not be empty for unhealthy status")
	}
}

func TestProvider_ReturnsNonNil(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if m.Provider() == nil {
		t.Error("Provider() returned nil after Init")
	}
}

// mockHeartbeat returns an httptest server that responds 200 OK on GET /.
func mockHeartbeat(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}
