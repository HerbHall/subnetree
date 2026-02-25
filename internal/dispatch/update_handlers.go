package dispatch

import (
	"fmt"
	"net/http"

	"github.com/HerbHall/subnetree/internal/version"
)

// UpdateManifest is the response for GET /dispatch/updates/latest.
type UpdateManifest struct {
	Version      string                    `json:"version"`
	Channel      string                    `json:"channel"`
	Platforms    map[string]PlatformBinary `json:"platforms"`
	ChecksumsURL string                   `json:"checksums_url"`
}

// PlatformBinary holds download info for one platform/arch combination.
type PlatformBinary struct {
	URL string `json:"url"`
}

// handleGetUpdateManifest returns the latest Scout release manifest.
//
//	@Summary		Get update manifest
//	@Description	Returns the latest Scout binary version and download URLs for all platforms.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UpdateManifest
//	@Router			/dispatch/updates/latest [get]
func (m *Module) handleGetUpdateManifest(w http.ResponseWriter, _ *http.Request) {
	ver := version.Short()
	platforms := make(map[string]PlatformBinary)
	for platform, archs := range validPlatformArch {
		for arch := range archs {
			key := platform + "/" + arch
			platforms[key] = PlatformBinary{
				URL: buildBinaryURL(platform, arch),
			}
		}
	}

	checksumURL := fmt.Sprintf(
		"https://github.com/HerbHall/subnetree/releases/download/v%s/checksums.txt", ver)
	if ver == "dev" {
		checksumURL = "https://github.com/HerbHall/subnetree/releases/latest/download/checksums.txt"
	}

	dispatchWriteJSON(w, http.StatusOK, UpdateManifest{
		Version:      ver,
		Channel:      "stable",
		Platforms:    platforms,
		ChecksumsURL: checksumURL,
	})
}
