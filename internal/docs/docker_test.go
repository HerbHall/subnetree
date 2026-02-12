package docs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// sampleContainersJSON is a realistic Docker Engine /containers/json response.
var sampleContainersJSON = `[
	{
		"Id": "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
		"Names": ["/nginx-proxy"],
		"Image": "nginx:latest",
		"State": "running",
		"Status": "Up 3 hours",
		"Ports": [{"IP": "0.0.0.0", "PrivatePort": 80, "PublicPort": 8080, "Type": "tcp"}],
		"Mounts": [{"Source": "/data/nginx", "Destination": "/etc/nginx", "Mode": "ro", "RW": false}],
		"Labels": {"com.docker.compose.project": "homelab"}
	},
	{
		"Id": "deadbeef1234deadbeef1234deadbeef1234deadbeef1234deadbeef1234dead",
		"Names": ["/postgres-db"],
		"Image": "postgres:16",
		"State": "exited",
		"Status": "Exited (0) 2 hours ago",
		"Ports": [],
		"Mounts": [],
		"Labels": {}
	}
]`

// sampleInspectJSON is a realistic (abbreviated) Docker Engine /containers/{id}/json response.
var sampleInspectJSON = `{
	"Id": "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
	"Created": "2025-01-15T10:00:00Z",
	"Name": "/nginx-proxy",
	"State": {"Status": "running", "Running": true, "Pid": 12345},
	"Config": {"Image": "nginx:latest", "Env": ["NGINX_PORT=80"]},
	"HostConfig": {"RestartPolicy": {"Name": "always"}}
}`

func TestDockerCollector_Name(t *testing.T) {
	dc := NewDockerCollector("tcp://localhost:9999", zap.NewNop())
	if got := dc.Name(); got != "docker" {
		t.Errorf("Name() = %q, want %q", got, "docker")
	}
}

func TestDockerCollector_Available_Reachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())
	if !dc.Available() {
		t.Error("Available() = false, want true")
	}
}

func TestDockerCollector_Available_Unreachable(t *testing.T) {
	// Point to a port that is not listening.
	dc := NewDockerCollector("tcp://127.0.0.1:1", zap.NewNop())
	if dc.Available() {
		t.Error("Available() = true, want false")
	}
}

func TestDockerCollector_Discover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/containers/json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sampleContainersJSON))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())

	apps, err := dc.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(apps) != 2 {
		t.Fatalf("len(apps) = %d, want 2", len(apps))
	}

	// First container: running nginx.
	if apps[0].Name != "nginx-proxy" {
		t.Errorf("apps[0].Name = %q, want %q", apps[0].Name, "nginx-proxy")
	}
	if apps[0].AppType != "docker-container" {
		t.Errorf("apps[0].AppType = %q, want %q", apps[0].AppType, "docker-container")
	}
	if apps[0].Status != "active" {
		t.Errorf("apps[0].Status = %q, want %q", apps[0].Status, "active")
	}
	if apps[0].Collector != "docker" {
		t.Errorf("apps[0].Collector = %q, want %q", apps[0].Collector, "docker")
	}

	// Second container: exited postgres.
	if apps[1].Name != "postgres-db" {
		t.Errorf("apps[1].Name = %q, want %q", apps[1].Name, "postgres-db")
	}
	if apps[1].Status != "inactive" {
		t.Errorf("apps[1].Status = %q, want %q", apps[1].Status, "inactive")
	}

	// Verify metadata is valid JSON with expected fields.
	var meta map[string]any
	if err := json.Unmarshal([]byte(apps[0].Metadata), &meta); err != nil {
		t.Fatalf("apps[0].Metadata is not valid JSON: %v", err)
	}
	if meta["image"] != "nginx:latest" {
		t.Errorf("metadata.image = %v, want %q", meta["image"], "nginx:latest")
	}
}

func TestDockerCollector_Discover_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/containers/json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())

	apps, err := dc.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("len(apps) = %d, want 0", len(apps))
	}
}

func TestDockerCollector_Discover_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"daemon error"}`))
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())

	_, err := dc.Discover(context.Background())
	if err == nil {
		t.Fatal("Discover() error = nil, want error")
	}
}

func TestDockerCollector_Collect(t *testing.T) {
	const containerID = "abc123def456"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/containers/"+containerID+"/json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sampleInspectJSON))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())

	cfg, err := dc.Collect(context.Background(), containerID)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if cfg.Format != "json" {
		t.Errorf("Format = %q, want %q", cfg.Format, "json")
	}
	if cfg.Content == "" {
		t.Error("Content is empty, want non-empty JSON")
	}

	// Verify the content is valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(cfg.Content), &parsed); err != nil {
		t.Fatalf("Content is not valid JSON: %v", err)
	}
	if parsed["Name"] != "/nginx-proxy" {
		t.Errorf("parsed Name = %v, want %q", parsed["Name"], "/nginx-proxy")
	}
}

func TestDockerCollector_Collect_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"No such container"}`))
	}))
	defer srv.Close()

	dc := NewDockerCollector("tcp://"+srv.Listener.Addr().String(), zap.NewNop())

	_, err := dc.Collect(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Collect() error = nil, want error for not-found container")
	}
}
