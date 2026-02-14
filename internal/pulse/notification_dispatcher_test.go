package pulse

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

func newTestDispatcher(t *testing.T) (*NotificationDispatcher, *PulseStore, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations.
	for _, m := range migrations() {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		if err := m.Up(tx); err != nil {
			tx.Rollback()
			t.Fatalf("migration %d: %v", m.Version, err)
		}
		tx.Commit()
	}

	store := NewPulseStore(db)
	dispatcher := NewNotificationDispatcher(store, zap.NewNop())
	return dispatcher, store, db
}

func TestNotificationDispatcher_HandleAlertEvent_Webhook(t *testing.T) {
	dispatcher, store, _ := newTestDispatcher(t)

	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Create webhook channel.
	cfgJSON, _ := json.Marshal(WebhookConfig{URL: srv.URL})
	ch := &NotificationChannel{
		ID:        "notif-wh-1",
		Name:      "Test Webhook",
		Type:      "webhook",
		Config:    string(cfgJSON),
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.InsertChannel(context.Background(), ch); err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	// Dispatch an alert triggered event.
	alert := &Alert{
		ID:       "alert-dispatch-1",
		CheckID:  "check-1",
		DeviceID: "device-1",
		Severity: "warning",
		Message:  "test alert",
	}
	event := plugin.Event{
		Topic:     TopicAlertTriggered,
		Source:    "pulse",
		Timestamp: time.Now().UTC(),
		Payload:   alert,
	}

	dispatcher.HandleAlertEvent(context.Background(), event)

	if receivedBody == nil {
		t.Fatal("webhook did not receive a request")
	}

	var payload webhookPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.EventType != "triggered" {
		t.Errorf("event_type = %q, want %q", payload.EventType, "triggered")
	}
	if payload.Alert.ID != "alert-dispatch-1" {
		t.Errorf("alert.id = %q, want %q", payload.Alert.ID, "alert-dispatch-1")
	}
}

func TestNotificationDispatcher_HandleAlertEvent_Resolved(t *testing.T) {
	dispatcher, store, _ := newTestDispatcher(t)

	var receivedPayload webhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(WebhookConfig{URL: srv.URL})
	ch := &NotificationChannel{
		ID:        "notif-wh-2",
		Name:      "Resolved Webhook",
		Type:      "webhook",
		Config:    string(cfgJSON),
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.InsertChannel(context.Background(), ch); err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	now := time.Now().UTC()
	alert := &Alert{
		ID:         "alert-resolved-1",
		Severity:   "warning",
		ResolvedAt: &now,
	}
	event := plugin.Event{
		Topic:   TopicAlertResolved,
		Payload: alert,
	}

	dispatcher.HandleAlertEvent(context.Background(), event)

	if receivedPayload.EventType != "resolved" {
		t.Errorf("event_type = %q, want %q", receivedPayload.EventType, "resolved")
	}
}

func TestNotificationDispatcher_HandleAlertEvent_DisabledChannelSkipped(t *testing.T) {
	dispatcher, store, _ := newTestDispatcher(t)

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(WebhookConfig{URL: srv.URL})
	ch := &NotificationChannel{
		ID:        "notif-wh-disabled",
		Name:      "Disabled Webhook",
		Type:      "webhook",
		Config:    string(cfgJSON),
		Enabled:   false, // disabled
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.InsertChannel(context.Background(), ch); err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	alert := &Alert{ID: "alert-skip"}
	event := plugin.Event{Topic: TopicAlertTriggered, Payload: alert}

	dispatcher.HandleAlertEvent(context.Background(), event)

	if called {
		t.Error("disabled channel should not have been called")
	}
}

func TestNotificationDispatcher_HandleAlertEvent_FailedNotificationDoesNotBlockOthers(t *testing.T) {
	dispatcher, store, _ := newTestDispatcher(t)

	// First server: always fails.
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()

	// Second server: succeeds.
	secondCalled := false
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secondCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	now := time.Now().UTC()
	failCfg, _ := json.Marshal(WebhookConfig{URL: failSrv.URL})
	okCfg, _ := json.Marshal(WebhookConfig{URL: okSrv.URL})

	// Insert fail channel first (earlier created_at ensures ordering).
	if err := store.InsertChannel(context.Background(), &NotificationChannel{
		ID: "notif-fail", Name: "Fail", Type: "webhook",
		Config: string(failCfg), Enabled: true,
		CreatedAt: now.Add(-time.Second), UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert fail channel: %v", err)
	}
	if err := store.InsertChannel(context.Background(), &NotificationChannel{
		ID: "notif-ok", Name: "OK", Type: "webhook",
		Config: string(okCfg), Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert ok channel: %v", err)
	}

	alert := &Alert{ID: "alert-multi"}
	event := plugin.Event{Topic: TopicAlertTriggered, Payload: alert}

	dispatcher.HandleAlertEvent(context.Background(), event)

	if !secondCalled {
		t.Error("second channel should have been called despite first failing")
	}
}

func TestNotificationDispatcher_HandleAlertEvent_NoChannels(t *testing.T) {
	dispatcher, _, _ := newTestDispatcher(t)

	alert := &Alert{ID: "alert-noop"}
	event := plugin.Event{Topic: TopicAlertTriggered, Payload: alert}

	// Should not panic or error.
	dispatcher.HandleAlertEvent(context.Background(), event)
}
