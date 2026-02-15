package pulse

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAlertmanagerNotifier_Notify_Firing(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewAlertmanagerNotifier(AlertmanagerConfig{URL: srv.URL})

	triggeredAt := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	alert := &Alert{
		ID:          "alert-am-1",
		CheckID:     "check-ping",
		DeviceID:    "device-router",
		Severity:    "critical",
		Message:     "ping failed",
		TriggeredAt: triggeredAt,
	}

	if err := notifier.Notify(t.Context(), alert, "triggered"); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if receivedBody == nil {
		t.Fatal("server did not receive a request")
	}

	// Verify headers.
	if ct := receivedHeaders.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if ua := receivedHeaders.Get("User-Agent"); ua != "SubNetree-Alertmanager/0.1" {
		t.Errorf("User-Agent = %q, want %q", ua, "SubNetree-Alertmanager/0.1")
	}

	var payload alertmanagerPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Version != "4" {
		t.Errorf("version = %q, want %q", payload.Version, "4")
	}
	if payload.Status != "firing" {
		t.Errorf("status = %q, want %q", payload.Status, "firing")
	}
	if len(payload.Alerts) != 1 {
		t.Fatalf("alerts count = %d, want 1", len(payload.Alerts))
	}

	am := payload.Alerts[0]
	if am.Status != "firing" {
		t.Errorf("alert.status = %q, want %q", am.Status, "firing")
	}

	// Verify labels.
	expectedLabels := map[string]string{
		"alertname": "SubNetreeAlert",
		"device_id": "device-router",
		"check_id":  "check-ping",
		"severity":  "critical",
		"source":    "subnetree",
	}
	for k, want := range expectedLabels {
		if got := am.Labels[k]; got != want {
			t.Errorf("labels[%q] = %q, want %q", k, got, want)
		}
	}

	// Verify annotations.
	if got := am.Annotations["summary"]; got != "ping failed" {
		t.Errorf("annotations[summary] = %q, want %q", got, "ping failed")
	}

	// Verify startsAt matches triggeredAt.
	if !am.StartsAt.Equal(triggeredAt) {
		t.Errorf("startsAt = %v, want %v", am.StartsAt, triggeredAt)
	}

	// Verify endsAt is zero for firing alert.
	if !am.EndsAt.IsZero() {
		t.Errorf("endsAt = %v, want zero value", am.EndsAt)
	}
}

func TestAlertmanagerNotifier_Notify_Resolved(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewAlertmanagerNotifier(AlertmanagerConfig{URL: srv.URL})

	triggeredAt := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	resolvedAt := time.Date(2026, 2, 15, 10, 5, 0, 0, time.UTC)
	alert := &Alert{
		ID:          "alert-am-2",
		CheckID:     "check-http",
		DeviceID:    "device-web",
		Severity:    "warning",
		Message:     "HTTP 503",
		TriggeredAt: triggeredAt,
		ResolvedAt:  &resolvedAt,
	}

	if err := notifier.Notify(t.Context(), alert, "resolved"); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	var payload alertmanagerPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Status != "resolved" {
		t.Errorf("status = %q, want %q", payload.Status, "resolved")
	}

	am := payload.Alerts[0]
	if am.Status != "resolved" {
		t.Errorf("alert.status = %q, want %q", am.Status, "resolved")
	}
	if !am.EndsAt.Equal(resolvedAt) {
		t.Errorf("endsAt = %v, want %v", am.EndsAt, resolvedAt)
	}
}

func TestAlertmanagerNotifier_Notify_HMAC(t *testing.T) {
	var receivedBody []byte
	var receivedSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSig = r.Header.Get("X-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	secret := "my-webhook-secret"
	notifier := NewAlertmanagerNotifier(AlertmanagerConfig{
		URL:    srv.URL,
		Secret: secret,
	})

	alert := &Alert{
		ID:       "alert-am-3",
		CheckID:  "check-1",
		DeviceID: "device-1",
		Severity: "info",
		Message:  "test hmac",
	}

	if err := notifier.Notify(t.Context(), alert, "triggered"); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if receivedSig == "" {
		t.Fatal("X-Signature header not present")
	}

	// Verify signature matches expected HMAC-SHA256.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expected {
		t.Errorf("X-Signature = %q, want %q", receivedSig, expected)
	}
}

func TestAlertmanagerNotifier_Notify_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	notifier := NewAlertmanagerNotifier(AlertmanagerConfig{URL: srv.URL})

	alert := &Alert{
		ID:       "alert-am-4",
		Severity: "warning",
		Message:  "test error",
	}

	err := notifier.Notify(t.Context(), alert, "triggered")
	if err == nil {
		t.Fatal("expected error for HTTP 503, got nil")
	}
}

func TestAlertmanagerNotifier_Type(t *testing.T) {
	notifier := NewAlertmanagerNotifier(AlertmanagerConfig{URL: "http://localhost"})
	if got := notifier.Type(); got != "alertmanager" {
		t.Errorf("Type() = %q, want %q", got, "alertmanager")
	}
}
