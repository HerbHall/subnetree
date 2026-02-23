package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoMiddleware(t *testing.T) {
	// Backend handler that always returns 200 OK.
	backend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := DemoMiddleware(backend)

	tests := []struct {
		name           string
		method         string
		wantStatus     int
		wantPassThru   bool
		wantDemoError  bool
		wantJSONHeader bool
	}{
		{name: "GET passes through", method: http.MethodGet, wantStatus: http.StatusOK, wantPassThru: true},
		{name: "HEAD passes through", method: http.MethodHead, wantStatus: http.StatusOK, wantPassThru: true},
		{name: "OPTIONS passes through", method: http.MethodOptions, wantStatus: http.StatusOK, wantPassThru: true},
		{name: "POST blocked", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed, wantDemoError: true, wantJSONHeader: true},
		{name: "PUT blocked", method: http.MethodPut, wantStatus: http.StatusMethodNotAllowed, wantDemoError: true, wantJSONHeader: true},
		{name: "DELETE blocked", method: http.MethodDelete, wantStatus: http.StatusMethodNotAllowed, wantDemoError: true, wantJSONHeader: true},
		{name: "PATCH blocked", method: http.MethodPatch, wantStatus: http.StatusMethodNotAllowed, wantDemoError: true, wantJSONHeader: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/test", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body, _ := io.ReadAll(w.Result().Body)
			bodyStr := string(body)

			if tc.wantPassThru && !strings.Contains(bodyStr, `"status":"ok"`) {
				t.Errorf("expected backend response, got %q", bodyStr)
			}

			if tc.wantDemoError && !strings.Contains(bodyStr, "demo mode") {
				t.Errorf("expected 'demo mode' in body, got %q", bodyStr)
			}

			if tc.wantJSONHeader {
				ct := w.Header().Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
		})
	}
}
