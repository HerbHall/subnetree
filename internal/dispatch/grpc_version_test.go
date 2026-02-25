package dispatch

import (
	"testing"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/version"
	"go.uber.org/zap"
)

func TestCheckAgentVersion(t *testing.T) {
	origVersion := version.Version
	defer func() { version.Version = origVersion }()

	s := &scoutServer{logger: zap.NewNop()}

	tests := []struct {
		name          string
		serverVersion string
		agentVersion  string
		want          scoutpb.VersionStatus
	}{
		{
			name:          "agent behind server",
			serverVersion: "1.0.0",
			agentVersion:  "0.5.0",
			want:          scoutpb.VersionStatus_VERSION_UPDATE_AVAILABLE,
		},
		{
			name:          "agent matches server",
			serverVersion: "1.0.0",
			agentVersion:  "1.0.0",
			want:          scoutpb.VersionStatus_VERSION_OK,
		},
		{
			name:          "agent ahead of server",
			serverVersion: "1.0.0",
			agentVersion:  "1.1.0",
			want:          scoutpb.VersionStatus_VERSION_OK,
		},
		{
			name:          "agent is dev",
			serverVersion: "1.0.0",
			agentVersion:  "dev",
			want:          scoutpb.VersionStatus_VERSION_OK,
		},
		{
			name:          "server is dev",
			serverVersion: "dev",
			agentVersion:  "0.5.0",
			want:          scoutpb.VersionStatus_VERSION_OK,
		},
		{
			name:          "both dev",
			serverVersion: "dev",
			agentVersion:  "dev",
			want:          scoutpb.VersionStatus_VERSION_OK,
		},
		{
			name:          "with v prefix",
			serverVersion: "v1.0.0",
			agentVersion:  "v0.9.0",
			want:          scoutpb.VersionStatus_VERSION_UPDATE_AVAILABLE,
		},
		{
			name:          "patch behind",
			serverVersion: "1.0.1",
			agentVersion:  "1.0.0",
			want:          scoutpb.VersionStatus_VERSION_UPDATE_AVAILABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version.Version = tt.serverVersion
			got := s.checkAgentVersion(tt.agentVersion)
			if got != tt.want {
				t.Errorf("checkAgentVersion(%q) with server %q = %v, want %v",
					tt.agentVersion, tt.serverVersion, got, tt.want)
			}
		})
	}
}

func TestNormalizeSemver(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"dev", "vdev"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeSemver(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSemver(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSplitPlatformArch(t *testing.T) {
	tests := []struct {
		input    string
		wantOS   string
		wantArch string
	}{
		{"linux/amd64", "linux", "amd64"},
		{"windows/amd64", "windows", "amd64"},
		{"darwin/arm64", "darwin", "arm64"},
		{"linux", "linux", "amd64"},
	}
	for _, tt := range tests {
		goos, goarch := splitPlatformArch(tt.input)
		if goos != tt.wantOS || goarch != tt.wantArch {
			t.Errorf("splitPlatformArch(%q) = (%q, %q), want (%q, %q)",
				tt.input, goos, goarch, tt.wantOS, tt.wantArch)
		}
	}
}
