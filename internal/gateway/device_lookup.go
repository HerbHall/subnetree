package gateway

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
)

// DeviceLookup is the consumer-side interface for device resolution.
// Defined where consumed (gateway) rather than where implemented (recon),
// following the consumer-side interface convention.
type DeviceLookup interface {
	DeviceByID(ctx context.Context, id string) (*models.Device, error)
}

// resolveDeviceLookup attempts to find a plugin filling the "discovery" role
// that also implements DeviceLookup. Returns nil and an error if unavailable.
func resolveDeviceLookup(resolver plugin.PluginResolver) (DeviceLookup, error) {
	if resolver == nil {
		return nil, fmt.Errorf("plugin resolver not available")
	}

	plugins := resolver.ResolveByRole(roles.RoleDiscovery)
	if len(plugins) == 0 {
		return nil, fmt.Errorf("no plugin with role %q registered", roles.RoleDiscovery)
	}

	for _, p := range plugins {
		if dl, ok := p.(DeviceLookup); ok {
			return dl, nil
		}
	}

	return nil, fmt.Errorf("discovery plugin does not implement DeviceLookup")
}
