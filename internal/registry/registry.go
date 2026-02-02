// Package registry manages plugin lifecycle: registration, dependency resolution,
// initialization, and shutdown of NetVantage plugins.
package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// Registry manages the lifecycle of all registered plugins.
type Registry struct {
	mu       sync.RWMutex
	plugins  map[string]plugin.Plugin
	infos    map[string]plugin.PluginInfo
	order    []string // topological order after Validate
	disabled map[string]bool
	logger   *zap.Logger
}

// New creates a new plugin registry.
func New(logger *zap.Logger) *Registry {
	return &Registry{
		plugins:  make(map[string]plugin.Plugin),
		infos:    make(map[string]plugin.PluginInfo),
		disabled: make(map[string]bool),
		logger:   logger,
	}
}

// Register adds a plugin to the registry. Must be called before Validate.
func (r *Registry) Register(p plugin.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := p.Info()
	name := info.Name

	if name == "" {
		return fmt.Errorf("plugin has empty name")
	}
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	r.infos[name] = info
	r.logger.Info("plugin registered",
		zap.String("name", name),
		zap.String("version", info.Version),
		zap.Int("api_version", info.APIVersion),
	)
	return nil
}

// Validate checks API version compatibility, resolves dependencies via
// topological sort, and verifies there are no cycles or missing dependencies.
func (r *Registry) Validate() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check API version compatibility for each plugin.
	for name, info := range r.infos {
		if err := r.checkAPIVersion(name, info.APIVersion); err != nil {
			if info.Required {
				return err
			}
			r.logger.Warn("disabling plugin due to API version incompatibility",
				zap.String("name", name),
				zap.Error(err),
			)
			r.disabled[name] = true
		}
	}

	// Check that all dependencies exist and are not disabled.
	for name, info := range r.infos {
		if r.disabled[name] {
			continue
		}
		for _, dep := range info.Dependencies {
			if _, ok := r.plugins[dep]; !ok {
				if info.Required {
					return fmt.Errorf("plugin %q depends on %q which is not registered", name, dep)
				}
				r.logger.Warn("disabling plugin due to missing dependency",
					zap.String("name", name),
					zap.String("missing_dep", dep),
				)
				r.disabled[name] = true
				break
			}
			if r.disabled[dep] {
				if info.Required {
					return fmt.Errorf("plugin %q depends on %q which is disabled", name, dep)
				}
				r.logger.Warn("disabling plugin: dependency is disabled",
					zap.String("name", name),
					zap.String("disabled_dep", dep),
				)
				r.disabled[name] = true
				break
			}
		}
	}

	// Cascade disable: if a plugin is disabled, disable all its dependents.
	changed := true
	for changed {
		changed = false
		for name, info := range r.infos {
			if r.disabled[name] {
				continue
			}
			for _, dep := range info.Dependencies {
				if !r.disabled[dep] {
					continue
				}
				if info.Required {
					return fmt.Errorf("required plugin %q cannot start: dependency %q is disabled", name, dep)
				}
				r.logger.Warn("cascade disabling plugin",
					zap.String("name", name),
					zap.String("disabled_dep", dep),
				)
				r.disabled[name] = true
				changed = true
				break
			}
		}
	}

	// Topological sort of active plugins.
	order, err := r.topologicalSort()
	if err != nil {
		return err
	}
	r.order = order

	r.logger.Info("plugin dependency resolution complete",
		zap.Strings("start_order", r.order),
		zap.Int("active", len(r.order)),
		zap.Int("disabled", len(r.disabled)),
	)
	return nil
}

// InitAll initializes all active plugins in dependency order.
func (r *Registry) InitAll(ctx context.Context, depsFn func(name string) plugin.Dependencies) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		p := r.plugins[name]

		// Validate config if plugin implements Validator.
		r.logger.Info("initializing plugin", zap.String("name", name))
		deps := depsFn(name)
		if err := p.Init(ctx, deps); err != nil {
			info := r.infos[name]
			if info.Required {
				return fmt.Errorf("required plugin %q failed to initialize: %w", name, err)
			}
			r.logger.Error("optional plugin failed to initialize, disabling",
				zap.String("name", name),
				zap.Error(err),
			)
			r.disabled[name] = true
			continue
		}

		// Post-init config validation for Validator plugins.
		if v, ok := p.(plugin.Validator); ok {
			if err := v.ValidateConfig(); err != nil {
				info := r.infos[name]
				if info.Required {
					return fmt.Errorf("required plugin %q config validation failed: %w", name, err)
				}
				r.logger.Error("optional plugin config validation failed, disabling",
					zap.String("name", name),
					zap.Error(err),
				)
				r.disabled[name] = true
			}
		}
	}
	return nil
}

// StartAll starts all initialized plugins in dependency order.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		if r.disabled[name] {
			continue
		}
		p := r.plugins[name]
		r.logger.Info("starting plugin", zap.String("name", name))
		if err := p.Start(ctx); err != nil {
			info := r.infos[name]
			if info.Required {
				return fmt.Errorf("required plugin %q failed to start: %w", name, err)
			}
			r.logger.Error("optional plugin failed to start, disabling",
				zap.String("name", name),
				zap.Error(err),
			)
			r.disabled[name] = true
		}
	}
	return nil
}

// StopAll stops all active plugins in reverse dependency order.
func (r *Registry) StopAll(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		if r.disabled[name] {
			continue
		}
		p := r.plugins[name]
		r.logger.Info("stopping plugin", zap.String("name", name))
		if err := p.Stop(ctx); err != nil {
			r.logger.Error("failed to stop plugin", zap.String("name", name), zap.Error(err))
		}
	}
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (plugin.Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	if ok && r.disabled[name] {
		return nil, false
	}
	return p, ok
}

// All returns all active (non-disabled) plugins in dependency order.
func (r *Registry) All() []plugin.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]plugin.Plugin, 0, len(r.order))
	for _, name := range r.order {
		if !r.disabled[name] {
			result = append(result, r.plugins[name])
		}
	}
	return result
}

// AllRoutes returns HTTP routes from all active plugins implementing HTTPProvider.
func (r *Registry) AllRoutes() map[string][]plugin.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make(map[string][]plugin.Route)
	for _, name := range r.order {
		if r.disabled[name] {
			continue
		}
		p := r.plugins[name]
		if hp, ok := p.(plugin.HTTPProvider); ok {
			if pr := hp.Routes(); len(pr) > 0 {
				routes[name] = pr
			}
		}
	}
	return routes
}

// Resolve returns a plugin by name (implements plugin.PluginResolver).
func (r *Registry) Resolve(name string) (plugin.Plugin, bool) {
	return r.Get(name)
}

// ResolveByRole returns all active plugins that declare the given role.
func (r *Registry) ResolveByRole(role string) []plugin.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []plugin.Plugin
	for _, name := range r.order {
		if r.disabled[name] {
			continue
		}
		info := r.infos[name]
		for _, pluginRole := range info.Roles {
			if pluginRole == role {
				result = append(result, r.plugins[name])
				break
			}
		}
	}
	return result
}

// IsDisabled returns whether a plugin has been disabled.
func (r *Registry) IsDisabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name]
}

// checkAPIVersion validates a plugin's API version against the server's range.
func (r *Registry) checkAPIVersion(name string, apiVersion int) error {
	if apiVersion < plugin.APIVersionMin {
		return fmt.Errorf(
			"plugin %q targets Plugin API v%d, but this server requires v%d or newer (current: v%d). Upgrade the plugin or use an older server",
			name, apiVersion, plugin.APIVersionMin, plugin.APIVersionCurrent,
		)
	}
	if apiVersion > plugin.APIVersionCurrent {
		return fmt.Errorf(
			"plugin %q targets Plugin API v%d, but this server only supports up to v%d. Upgrade the server to use this plugin",
			name, apiVersion, plugin.APIVersionCurrent,
		)
	}
	if apiVersion < plugin.APIVersionCurrent {
		r.logger.Warn(
			fmt.Sprintf("plugin %q targets Plugin API v%d (server current: v%d). The plugin will work but may not support newer features",
				name, apiVersion, plugin.APIVersionCurrent),
		)
	}
	return nil
}

// topologicalSort returns plugin names in dependency order using Kahn's algorithm.
func (r *Registry) topologicalSort() ([]string, error) {
	// Build adjacency list and in-degree count for active plugins only.
	active := make(map[string]bool)
	for name := range r.plugins {
		if !r.disabled[name] {
			active[name] = true
		}
	}

	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep -> list of plugins that depend on it

	for name := range active {
		inDegree[name] = 0
	}

	for name := range active {
		info := r.infos[name]
		for _, dep := range info.Dependencies {
			if active[dep] {
				inDegree[name]++
				dependents[dep] = append(dependents[dep], name)
			}
		}
	}

	// Start with plugins that have no dependencies.
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		order = append(order, name)

		for _, dependent := range dependents[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(order) != len(active) {
		// Find the cycle for a useful error message.
		var cycled []string
		for name := range active {
			if inDegree[name] > 0 {
				cycled = append(cycled, name)
			}
		}
		return nil, fmt.Errorf("dependency cycle detected among plugins: %v", cycled)
	}

	return order, nil
}
