package gateway

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// --- Mock TokenValidator ---

type mockTokenValidator struct {
	userID string
	err    error
}

func (m *mockTokenValidator) ValidateAccessToken(_ string) (*TokenClaims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &TokenClaims{UserID: m.userID}, nil
}

// --- Test SSH Server ---

// generateTestHostKey generates an ed25519 host key for the test SSH server.
func generateTestHostKey(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return signer
}

// newTestSSHServer starts an in-process SSH server that accepts password auth
// and echoes back whatever is sent to the session's stdin.
func newTestSSHServer(t *testing.T, username, password string) (addr string, cleanup func()) {
	t.Helper()

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == username && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(generateTestHostKey(t))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleTestSSHConn(conn, config)
		}
	}()

	return listener.Addr().String(), func() {
		listener.Close()
		<-done
	}
}

func handleTestSSHConn(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}

		go func() {
			for req := range requests {
				switch req.Type {
				case "pty-req", "shell":
					if req.WantReply {
						req.Reply(true, nil)
					}
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}()

		// Echo back whatever is sent.
		go func() {
			defer channel.Close()
			_, _ = io.Copy(channel, channel)
		}()
	}
}

// --- SSHBridge Tests ---

func newTestBridge(t *testing.T, validator TokenValidator) (*SSHBridge, *Module) {
	t.Helper()
	m := newTestModule(t)
	bridge := &SSHBridge{
		module: m,
		tokens: validator,
		logger: zap.NewNop(),
	}
	return bridge, m
}

// newTestSSHHTTPServer wraps the bridge handler in a mux with the correct
// route pattern so r.PathValue("device_id") works in tests.
func newTestSSHHTTPServer(t *testing.T, bridge *SSHBridge) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ws/gateway/ssh/{device_id}", bridge.HandleSSH)
	return httptest.NewServer(mux)
}

// sshWSURL constructs the WebSocket URL for connecting to a test SSH server.
func sshWSURL(srvURL, deviceID string, params map[string]string) string {
	base := "ws" + strings.TrimPrefix(srvURL, "http") +
		"/api/v1/ws/gateway/ssh/" + deviceID
	parts := make([]string, 0, len(params))
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	if len(parts) > 0 {
		base += "?" + strings.Join(parts, "&")
	}
	return base
}

// TestSSHBridge_MissingToken verifies that requests without a token are rejected.
func TestSSHBridge_MissingToken(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	bridge.HandleSSH(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rr.Body.String(), "missing token") {
		t.Errorf("body = %q, want to contain %q", rr.Body.String(), "missing token")
	}
}

// TestSSHBridge_InvalidToken verifies that requests with an invalid token are rejected.
func TestSSHBridge_InvalidToken(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{err: fmt.Errorf("bad token")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/dev-1?token=bad", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	bridge.HandleSSH(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rr.Body.String(), "invalid or expired token") {
		t.Errorf("body = %q, want to contain %q", rr.Body.String(), "invalid or expired token")
	}
}

// TestSSHBridge_MissingDeviceID verifies that requests without device_id are rejected.
func TestSSHBridge_MissingDeviceID(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/?token=valid", http.NoBody)
	// No SetPathValue for device_id -- simulates empty path value.
	rr := httptest.NewRecorder()

	bridge.HandleSSH(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "device_id is required") {
		t.Errorf("body = %q, want to contain %q", rr.Body.String(), "device_id is required")
	}
}

// TestSSHBridge_NoHostResolution verifies that requests fail when no host can be resolved.
func TestSSHBridge_NoHostResolution(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/dev-1?token=valid", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	bridge.HandleSSH(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "unable to resolve device address") {
		t.Errorf("body = %q, want to contain %q", rr.Body.String(), "unable to resolve device address")
	}
}

// TestSSHBridge_NilSessions verifies the handler returns 503 when sessions manager is nil.
func TestSSHBridge_NilSessions(t *testing.T) {
	validator := &mockTokenValidator{userID: "user-1"}
	m := &Module{
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
		ctx:    context.Background(),
		// sessions is nil
	}
	bridge := &SSHBridge{
		module: m,
		tokens: validator,
		logger: zap.NewNop(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/dev-1?token=valid&host=192.168.1.1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	bridge.HandleSSH(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// TestSSHBridge_CustomPort verifies that the port query parameter is respected.
func TestSSHBridge_CustomPort(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	var dialedAddr string
	bridge.sshDial = func(_, addr string, _ *ssh.ClientConfig) (*ssh.Client, error) {
		dialedAddr = addr
		return nil, fmt.Errorf("connection refused")
	}

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": "10.0.0.1", "port": "2222",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send credentials.
	creds, _ := json.Marshal(sshCredentials{Username: "admin", Password: "pass"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	// Wait for close (SSH dial fails).
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected read error after SSH dial failure")
	}

	if dialedAddr != "10.0.0.1:2222" {
		t.Errorf("dialed addr = %q, want %q", dialedAddr, "10.0.0.1:2222")
	}
}

// TestSSHBridge_SSHDialError verifies WebSocket is closed with error on SSH dial failure.
func TestSSHBridge_SSHDialError(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})
	bridge.sshDial = func(_, _ string, _ *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, fmt.Errorf("connection refused")
	}

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": "10.0.0.1",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send credentials.
	creds, _ := json.Marshal(sshCredentials{Username: "admin", Password: "pass"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	// The server should close the WebSocket with an error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected read error after SSH dial failure")
	}

	// Verify no session was created (SSH dial failed before session creation).
	if bridge.module.sessions.Count() != 0 {
		t.Errorf("session count = %d, want 0 after failed SSH dial", bridge.module.sessions.Count())
	}
}

// TestSSHBridge_InvalidCredentialsJSON verifies WebSocket is closed on bad JSON.
func TestSSHBridge_InvalidCredentialsJSON(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": "10.0.0.1",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send invalid JSON.
	if err := conn.Write(ctx, websocket.MessageText, []byte("not json")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Server should close with policy violation.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected read error after invalid JSON")
	}
}

// TestSSHBridge_EmptyUsername verifies WebSocket is closed when username is empty.
func TestSSHBridge_EmptyUsername(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": "10.0.0.1",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send credentials with empty username.
	creds, _ := json.Marshal(sshCredentials{Username: "", Password: "pass"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected read error after empty username")
	}
}

// TestSSHBridge_FullSession tests a complete SSH session lifecycle with the test SSH server.
func TestSSHBridge_FullSession(t *testing.T) {
	sshAddr, cleanup := newTestSSHServer(t, "admin", "secret")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(sshAddr)

	bridge, m := newTestBridge(t, &mockTokenValidator{userID: "user-42"})
	bus := &testEventBus{}
	m.bus = bus

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": host, "port": portStr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send credentials.
	creds, _ := json.Marshal(sshCredentials{Username: "admin", Password: "secret"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	// Verify session was created.
	time.Sleep(200 * time.Millisecond)
	if m.sessions.Count() != 1 {
		t.Errorf("session count = %d, want 1", m.sessions.Count())
	}

	// Verify audit entry was created.
	entries, err := m.store.ListAuditEntries(context.Background(), "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) < 1 {
		t.Fatal("expected at least 1 audit entry for session creation")
	}
	if entries[0].Action != "created" {
		t.Errorf("audit action = %q, want %q", entries[0].Action, "created")
	}
	if entries[0].SessionType != string(SessionTypeSSH) {
		t.Errorf("audit session_type = %q, want %q", entries[0].SessionType, string(SessionTypeSSH))
	}

	// Send data through the SSH echo server.
	testData := []byte("hello ssh")
	if err := conn.Write(ctx, websocket.MessageBinary, testData); err != nil {
		t.Fatalf("write test data: %v", err)
	}

	// Read echoed response.
	_, echoData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(echoData) != "hello ssh" {
		t.Errorf("echo = %q, want %q", string(echoData), "hello ssh")
	}

	// Close the WebSocket to trigger cleanup.
	conn.Close(websocket.StatusNormalClosure, "done")

	// Wait for cleanup.
	time.Sleep(300 * time.Millisecond)

	// Session should be cleaned up.
	if m.sessions.Count() != 0 {
		t.Errorf("session count = %d, want 0 after disconnect", m.sessions.Count())
	}

	// Verify close event was published.
	if e := bus.lastEvent(); e == nil || e.Topic != TopicSessionClosed {
		topic := "<nil>"
		if e != nil {
			topic = e.Topic
		}
		t.Errorf("last event topic = %q, want %q", topic, TopicSessionClosed)
	}
}

// TestSSHBridge_SessionCreatedEvent verifies that session creation publishes the correct event.
func TestSSHBridge_SessionCreatedEvent(t *testing.T) {
	sshAddr, cleanup := newTestSSHServer(t, "admin", "secret")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(sshAddr)

	bridge, m := newTestBridge(t, &mockTokenValidator{userID: "user-1"})
	bus := &testEventBus{}
	m.bus = bus

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": host, "port": portStr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	creds, _ := json.Marshal(sshCredentials{Username: "admin", Password: "secret"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	// Wait for session creation.
	time.Sleep(200 * time.Millisecond)

	// Check that session created event was published.
	found := false
	bus.mu.Lock()
	for _, e := range bus.events {
		if e.Topic == TopicSessionCreated {
			payload, ok := e.Payload.(map[string]string)
			if ok && payload["session_type"] == string(SessionTypeSSH) {
				found = true
			}
		}
	}
	bus.mu.Unlock()

	if !found {
		t.Error("expected gateway.session.created event with session_type=ssh")
	}

	conn.Close(websocket.StatusNormalClosure, "done")
	time.Sleep(200 * time.Millisecond)
}

// TestSSHBridge_SSHAuthFailure tests that wrong credentials result in WebSocket close.
func TestSSHBridge_SSHAuthFailure(t *testing.T) {
	sshAddr, cleanup := newTestSSHServer(t, "admin", "secret")
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(sshAddr)

	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": host, "port": portStr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Send wrong password.
	creds, _ := json.Marshal(sshCredentials{Username: "admin", Password: "wrong"})
	if err := conn.Write(ctx, websocket.MessageText, creds); err != nil {
		t.Fatalf("write creds: %v", err)
	}

	// Server should close with error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected read error after SSH auth failure")
	}

	// No session should be left.
	if bridge.module.sessions.Count() != 0 {
		t.Errorf("session count = %d, want 0 after auth failure", bridge.module.sessions.Count())
	}
}

// TestSSHBridge_HostFromQuery verifies the ?host= fallback parameter works.
func TestSSHBridge_HostFromQuery(t *testing.T) {
	bridge, _ := newTestBridge(t, &mockTokenValidator{userID: "user-1"})

	var dialedAddr string
	bridge.sshDial = func(_, addr string, _ *ssh.ClientConfig) (*ssh.Client, error) {
		dialedAddr = addr
		return nil, fmt.Errorf("refused")
	}

	srv := newTestSSHHTTPServer(t, bridge)
	defer srv.Close()

	wsURL := sshWSURL(srv.URL, "dev-1", map[string]string{
		"token": "valid", "host": "172.16.0.50",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	creds, _ := json.Marshal(sshCredentials{Username: "user", Password: "pass"})
	_ = conn.Write(ctx, websocket.MessageText, creds)

	// Wait for SSH dial attempt.
	_, _, _ = conn.Read(ctx)

	if dialedAddr != "172.16.0.50:22" {
		t.Errorf("dialed addr = %q, want %q", dialedAddr, "172.16.0.50:22")
	}
}

// --- SSHWebSocketHandler Tests ---

func TestSSHWebSocketHandler_RegisterRoutes(t *testing.T) {
	m := &Module{
		logger:   zap.NewNop(),
		cfg:      DefaultConfig(),
		sessions: NewSessionManager(100),
		ctx:      context.Background(),
	}
	validator := &mockTokenValidator{userID: "user-1"}
	handler := NewSSHWebSocketHandler(m, validator, zap.NewNop())

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Verify the route is registered by making a request.
	// Without a token, it should return 401 (proving the handler is mounted).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/gateway/ssh/dev-1", http.NoBody)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (route should be mounted)", rr.Code, http.StatusUnauthorized)
	}
}

func TestNewSSHWebSocketHandler(t *testing.T) {
	m := &Module{
		logger:   zap.NewNop(),
		cfg:      DefaultConfig(),
		sessions: NewSessionManager(100),
		ctx:      context.Background(),
	}
	validator := &mockTokenValidator{userID: "user-1"}
	handler := NewSSHWebSocketHandler(m, validator, zap.NewNop())

	if handler == nil {
		t.Fatal("NewSSHWebSocketHandler returned nil")
	}
	if handler.bridge == nil {
		t.Fatal("bridge should not be nil")
	}
	if handler.bridge.module != m {
		t.Error("bridge.module should reference the provided module")
	}
	if handler.bridge.tokens != validator {
		t.Error("bridge.tokens should reference the provided validator")
	}
}
