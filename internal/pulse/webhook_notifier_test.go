package pulse

import (
	"context"
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

func TestWebhookNotifier_Notify_Success(t *testing.T) {
	var received webhookPayload
	var headers http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{URL: srv.URL})
	alert := &Alert{
		ID:       "alert-1",
		CheckID:  "check-1",
		DeviceID: "device-1",
		Severity: "warning",
		Message:  "check failed",
	}

	err := notifier.Notify(context.Background(), alert, "triggered")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.EventType != "triggered" {
		t.Errorf("event_type = %q, want %q", received.EventType, "triggered")
	}
	if received.Alert.ID != "alert-1" {
		t.Errorf("alert.id = %q, want %q", received.Alert.ID, "alert-1")
	}
	if headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", headers.Get("Content-Type"), "application/json")
	}
	if headers.Get("User-Agent") != "SubNetree-Webhook/0.1" {
		t.Errorf("User-Agent = %q, want %q", headers.Get("User-Agent"), "SubNetree-Webhook/0.1")
	}
}

func TestWebhookNotifier_Notify_HMACSignature(t *testing.T) {
	secret := "test-secret-key"
	var receivedSig string
	var receivedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL:    srv.URL,
		Secret: secret,
	})

	alert := &Alert{
		ID:       "alert-hmac",
		Severity: "critical",
		Message:  "test hmac",
	}

	err := notifier.Notify(context.Background(), alert, "resolved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedSig == "" {
		t.Fatal("expected X-Signature header, got empty")
	}

	// Verify HMAC.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expectedSig {
		t.Errorf("signature mismatch: got %q, want %q", receivedSig, expectedSig)
	}
}

func TestWebhookNotifier_Notify_CustomHeaders(t *testing.T) {
	var headers http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL: srv.URL,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})

	err := notifier.Notify(context.Background(), &Alert{ID: "a1"}, "triggered")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if headers.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", headers.Get("X-Custom-Header"), "custom-value")
	}
}

func TestWebhookNotifier_Notify_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{URL: srv.URL})

	err := notifier.Notify(context.Background(), &Alert{ID: "a1"}, "triggered")
	if err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}

func TestWebhookNotifier_Notify_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{URL: srv.URL})
	notifier.client.Timeout = 50 * time.Millisecond

	err := notifier.Notify(context.Background(), &Alert{ID: "a1"}, "triggered")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWebhookNotifier_Notify_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(WebhookConfig{URL: srv.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := notifier.Notify(ctx, &Alert{ID: "a1"}, "triggered")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestWebhookNotifier_Type(t *testing.T) {
	n := NewWebhookNotifier(WebhookConfig{URL: "http://example.com"})
	if n.Type() != "webhook" {
		t.Errorf("Type() = %q, want %q", n.Type(), "webhook")
	}
}
