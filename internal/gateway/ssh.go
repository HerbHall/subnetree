package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// TokenValidator is the consumer-side interface for JWT validation.
// Defined where consumed (gateway) rather than where implemented (auth),
// following the consumer-side interface convention.
type TokenValidator interface {
	ValidateAccessToken(token string) (*TokenClaims, error)
}

// TokenClaims holds the subset of JWT claims needed by the SSH bridge.
type TokenClaims struct {
	UserID string
}

// sshCredentials is the JSON payload sent as the first WebSocket message
// to provide authentication credentials for the SSH connection.
type sshCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SSHBridge handles WebSocket-to-SSH bridging.
type SSHBridge struct {
	module *Module
	tokens TokenValidator
	logger *zap.Logger

	// sshDial is the function used to establish SSH connections.
	// Defaults to ssh.Dial; overridden in tests.
	sshDial func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// HandleSSH upgrades an HTTP request to a WebSocket connection and bridges it
// to an SSH session on the target device.
func (b *SSHBridge) HandleSSH(w http.ResponseWriter, r *http.Request) {
	// 1. Validate JWT from query parameter.
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token parameter", http.StatusUnauthorized)
		return
	}

	claims, err := b.tokens.ValidateAccessToken(token)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	// 2. Extract device_id from path.
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	// 3. Extract optional port (default 22) and host fallback.
	port := 22
	if portStr := r.URL.Query().Get("port"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p <= 65535 {
			port = p
		}
	}

	// 4. Resolve device IP via module's deviceLookup, or accept ?host= as fallback.
	host := r.URL.Query().Get("host")
	if b.module.deviceLookup != nil {
		device, err := b.module.deviceLookup.DeviceByID(r.Context(), deviceID)
		if err == nil && device != nil && len(device.IPAddresses) > 0 {
			host = device.IPAddresses[0]
		}
	}
	if host == "" {
		http.Error(w, "unable to resolve device address; provide ?host= parameter", http.StatusBadRequest)
		return
	}

	// 5. Check session capacity.
	if b.module.sessions == nil {
		http.Error(w, "gateway not ready", http.StatusServiceUnavailable)
		return
	}

	// 6. Accept WebSocket connection.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		b.logger.Error("websocket accept failed", zap.Error(err))
		return
	}

	ctx := r.Context()

	// 7. Read first message: JSON with credentials.
	_, msg, err := conn.Read(ctx)
	if err != nil {
		b.logger.Debug("failed to read credentials from websocket", zap.Error(err))
		conn.Close(websocket.StatusPolicyViolation, "failed to read credentials")
		return
	}

	var creds sshCredentials
	if err := json.Unmarshal(msg, &creds); err != nil {
		conn.Close(websocket.StatusPolicyViolation, "invalid credentials JSON")
		return
	}
	if creds.Username == "" {
		conn.Close(websocket.StatusPolicyViolation, "username is required")
		return
	}

	// 8. Dial SSH.
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	sshConfig := &ssh.ClientConfig{
		User: creds.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(creds.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // G106: user-facing tool, host key verification is a future enhancement
		Timeout:         10 * time.Second,
	}

	dial := b.sshDial
	if dial == nil {
		dial = ssh.Dial
	}
	client, err := dial("tcp", addr, sshConfig)
	if err != nil {
		b.logger.Debug("ssh dial failed",
			zap.String("addr", addr),
			zap.Error(err),
		)
		conn.Close(websocket.StatusInternalError, "SSH connection failed: "+err.Error())
		return
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		conn.Close(websocket.StatusInternalError, "SSH session creation failed")
		return
	}

	// 9. Request PTY and start shell.
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 24, 80, modes); err != nil {
		session.Close()
		client.Close()
		conn.Close(websocket.StatusInternalError, "PTY request failed")
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		conn.Close(websocket.StatusInternalError, "stdin pipe failed")
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		conn.Close(websocket.StatusInternalError, "stdout pipe failed")
		return
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		conn.Close(websocket.StatusInternalError, "shell start failed")
		return
	}

	// 10. Create a gateway session.
	gwSession := &Session{
		ID:          generateSessionID(),
		DeviceID:    deviceID,
		UserID:      claims.UserID,
		SessionType: SessionTypeSSH,
		Target:      ProxyTarget{Host: host, Port: port},
		SourceIP:    r.RemoteAddr,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(b.module.cfg.SessionTimeout),
	}

	if err := b.module.sessions.Create(gwSession); err != nil {
		session.Close()
		client.Close()
		conn.Close(websocket.StatusInternalError, "session limit reached")
		return
	}

	// 11. Record audit entry and publish event.
	if b.module.store != nil {
		entry := &AuditEntry{
			SessionID:   gwSession.ID,
			DeviceID:    deviceID,
			UserID:      claims.UserID,
			SessionType: string(SessionTypeSSH),
			Target:      fmt.Sprintf("%s:%d", host, port),
			Action:      "created",
			SourceIP:    r.RemoteAddr,
			Timestamp:   time.Now().UTC(),
		}
		if err := b.module.store.InsertAuditEntry(ctx, entry); err != nil {
			b.logger.Warn("failed to write SSH session audit entry", zap.Error(err))
		}
	}

	b.module.publishEvent(TopicSessionCreated, map[string]string{
		"session_id":   gwSession.ID,
		"device_id":    deviceID,
		"session_type": string(SessionTypeSSH),
		"target":       fmt.Sprintf("%s:%d", host, port),
	})

	// 12. Bidirectional copy between WebSocket and SSH.
	done := make(chan struct{}, 2)

	// WS -> SSH stdin
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			gwSession.BytesIn.Add(int64(len(data)))
			if _, err := stdin.Write(data); err != nil {
				return
			}
		}
	}()

	// SSH stdout -> WS
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				gwSession.BytesOut.Add(int64(n))
				if wErr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); wErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					b.logger.Debug("ssh stdout read error", zap.Error(err))
				}
				return
			}
		}
	}()

	// Wait for either goroutine to finish, then clean up.
	<-done

	session.Close()
	client.Close()
	conn.Close(websocket.StatusNormalClosure, "session ended")

	// Remove session and audit.
	b.module.sessions.Delete(gwSession.ID)
	b.module.logSessionClosed(gwSession, "disconnected")
}
