// Package plugintest provides shared contract tests that verify any
// plugin.Plugin implementation behaves correctly. Every module's test
// file should call TestPluginContract to ensure conformance.
package plugintest

import (
	"context"
	"testing"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// TestPluginContract runs a suite of behavioral contract tests against
// any plugin.Plugin implementation. Call this from each module's _test.go:
//
//	func TestContract(t *testing.T) {
//	    plugintest.TestPluginContract(t, func() plugin.Plugin { return recon.New() })
//	}
func TestPluginContract(t *testing.T, factory func() plugin.Plugin) {
	t.Helper()

	t.Run("Info_returns_valid_metadata", func(t *testing.T) {
		p := factory()
		info := p.Info()
		if info.Name == "" {
			t.Error("Info().Name must not be empty")
		}
		if info.Version == "" {
			t.Error("Info().Version must not be empty")
		}
		if info.APIVersion < plugin.APIVersionMin {
			t.Errorf("Info().APIVersion = %d, below minimum %d", info.APIVersion, plugin.APIVersionMin)
		}
	})

	t.Run("Init_succeeds_with_valid_deps", func(t *testing.T) {
		p := factory()
		deps := testDeps(p.Info().Name)
		if err := p.Init(context.Background(), deps); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
	})

	t.Run("Start_after_Init", func(t *testing.T) {
		p := factory()
		deps := testDeps(p.Info().Name)
		p.Init(context.Background(), deps)
		if err := p.Start(context.Background()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		// Clean up.
		p.Stop(context.Background())
	})

	t.Run("Stop_without_Start_does_not_panic", func(t *testing.T) {
		p := factory()
		deps := testDeps(p.Info().Name)
		p.Init(context.Background(), deps)
		if err := p.Stop(context.Background()); err != nil {
			t.Fatalf("Stop() without Start error = %v", err)
		}
	})

	t.Run("Info_is_idempotent", func(t *testing.T) {
		p := factory()
		a := p.Info()
		b := p.Info()
		if a.Name != b.Name || a.Version != b.Version {
			t.Error("Info() must return consistent results")
		}
	})
}

func testDeps(name string) plugin.Dependencies {
	logger, _ := zap.NewDevelopment()
	return plugin.Dependencies{
		Logger: logger.Named(name),
	}
}
