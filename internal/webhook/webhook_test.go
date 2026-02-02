package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/internal/recon"
	"github.com/HerbHall/netvantage/pkg/plugin"
	"github.com/HerbHall/netvantage/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestSubscriptions_ReturnsExpectedTopics(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{Logger: zap.NewNop()}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	subs := m.Subscriptions()
	if len(subs) != 3 {
		t.Fatalf("Subscriptions() returned %d, want 3", len(subs))
	}

	topics := make(map[string]bool)
	for _, s := range subs {
		topics[s.Topic] = true
	}

	expected := []string{
		recon.TopicDeviceDiscovered,
		recon.TopicDeviceUpdated,
		recon.TopicDeviceLost,
	}
	for _, topic := range expected {
		if !topics[topic] {
			t.Errorf("missing subscription for topic %q", topic)
		}
	}
}

func TestHandleEvent_DeliversWebhook(t *testing.T) {
	var mu sync.Mutex
	var received []WebhookPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("User-Agent") != "NetVantage-Webhook/0.1" {
			t.Errorf("User-Agent = %q, want NetVantage-Webhook/0.1", r.Header.Get("User-Agent"))
		}
		var p WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := New()
	m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: &testConfig{values: map[string]any{
			"url":     srv.URL,
			"timeout": 5 * time.Second,
			"enabled": true,
		}},
	})

	m.handleEvent(context.Background(), plugin.Event{
		Topic:     recon.TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Payload:   map[string]string{"ip": "192.168.1.1"},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received %d webhooks, want 1", len(received))
	}
	if received[0].Event != recon.TopicDeviceDiscovered {
		t.Errorf("event = %q, want %q", received[0].Event, recon.TopicDeviceDiscovered)
	}
	if received[0].Source != "recon" {
		t.Errorf("source = %q, want recon", received[0].Source)
	}
}

func TestHandleEvent_SkipsWhenDisabled(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := New()
	m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: &testConfig{values: map[string]any{
			"url":     srv.URL,
			"enabled": false,
		}},
	})

	m.handleEvent(context.Background(), plugin.Event{
		Topic:     recon.TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Now(),
	})

	if called {
		t.Error("expected webhook NOT to be called when disabled")
	}
}

func TestHandleEvent_SkipsWhenNoURL(t *testing.T) {
	m := New()
	m.Init(context.Background(), plugin.Dependencies{Logger: zap.NewNop()})

	// Should not panic when URL is empty.
	m.handleEvent(context.Background(), plugin.Event{
		Topic:     recon.TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Now(),
	})
}

func TestHandleEvent_LogsOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := New()
	m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: &testConfig{values: map[string]any{
			"url": srv.URL,
		}},
	})

	// Should not panic; warning is logged.
	m.handleEvent(context.Background(), plugin.Event{
		Topic:     recon.TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   map[string]string{"test": "data"},
	})
}

// testConfig is a minimal plugin.Config for tests.
type testConfig struct {
	values map[string]any
}

func (c *testConfig) Unmarshal(_ any) error    { return nil }
func (c *testConfig) Get(key string) any       { return c.values[key] }
func (c *testConfig) GetString(key string) string {
	v, _ := c.values[key].(string)
	return v
}
func (c *testConfig) GetInt(key string) int {
	v, _ := c.values[key].(int)
	return v
}
func (c *testConfig) GetBool(key string) bool {
	v, _ := c.values[key].(bool)
	return v
}
func (c *testConfig) GetDuration(key string) time.Duration {
	v, _ := c.values[key].(time.Duration)
	return v
}
func (c *testConfig) IsSet(key string) bool {
	_, ok := c.values[key]
	return ok
}
func (c *testConfig) Sub(_ string) plugin.Config {
	return &testConfig{values: map[string]any{}}
}
