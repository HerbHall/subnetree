package dispatch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HerbHall/subnetree/internal/version"
	"go.uber.org/zap"
)

func TestHandleGetUpdateManifest(t *testing.T) {
	// Save and restore version.
	origVersion := version.Version
	defer func() { version.Version = origVersion }()

	tests := []struct {
		name          string
		serverVersion string
		wantChannel   string
		wantPlatforms int
	}{
		{
			name:          "dev version",
			serverVersion: "dev",
			wantChannel:   "stable",
			wantPlatforms: 5, // linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
		},
		{
			name:          "release version",
			serverVersion: "1.2.3",
			wantChannel:   "stable",
			wantPlatforms: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version.Version = tt.serverVersion
			m := &Module{logger: zap.NewNop()}

			req := httptest.NewRequest(http.MethodGet, "/dispatch/updates/latest", http.NoBody)
			w := httptest.NewRecorder()

			m.handleGetUpdateManifest(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}

			var manifest UpdateManifest
			if err := json.NewDecoder(w.Body).Decode(&manifest); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if manifest.Version != tt.serverVersion {
				t.Errorf("version = %q, want %q", manifest.Version, tt.serverVersion)
			}
			if manifest.Channel != tt.wantChannel {
				t.Errorf("channel = %q, want %q", manifest.Channel, tt.wantChannel)
			}
			if len(manifest.Platforms) != tt.wantPlatforms {
				t.Errorf("platforms count = %d, want %d", len(manifest.Platforms), tt.wantPlatforms)
			}
			if manifest.ChecksumsURL == "" {
				t.Error("checksums_url is empty")
			}

			// Verify specific platforms exist.
			for _, key := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"} {
				p, ok := manifest.Platforms[key]
				if !ok {
					t.Errorf("platform %q not found", key)
					continue
				}
				if p.URL == "" {
					t.Errorf("platform %q has empty URL", key)
				}
			}
		})
	}
}
