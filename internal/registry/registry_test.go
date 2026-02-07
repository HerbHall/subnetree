package registry

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// testPlugin is a minimal plugin for testing.
type testPlugin struct {
	info    plugin.PluginInfo
	initErr error
}

func newTestPlugin(name string, deps ...string) *testPlugin {
	return &testPlugin{
		info: plugin.PluginInfo{
			Name:         name,
			Version:      "1.0.0",
			Description:  "test plugin " + name,
			Dependencies: deps,
			APIVersion:   plugin.APIVersionCurrent,
		},
	}
}

func (p *testPlugin) Info() plugin.PluginInfo                             { return p.info }
func (p *testPlugin) Init(_ context.Context, _ plugin.Dependencies) error { return p.initErr }
func (p *testPlugin) Start(_ context.Context) error                       { return nil }
func (p *testPlugin) Stop(_ context.Context) error                        { return nil }

// shutdownPlugin tracks stop order and simulates configurable stop behavior.
type shutdownPlugin struct {
	info         plugin.PluginInfo
	stopDuration time.Duration // how long Stop() takes
	stopErr      error         // error to return from Stop()
	stopOrder    *[]string     // shared slice to record stop order
	stopCount    *int32        // atomic counter for stop calls
}

func newShutdownPlugin(name string, stopOrder *[]string, deps ...string) *shutdownPlugin {
	return &shutdownPlugin{
		info: plugin.PluginInfo{
			Name:         name,
			Version:      "1.0.0",
			Description:  "shutdown test plugin " + name,
			Dependencies: deps,
			APIVersion:   plugin.APIVersionCurrent,
		},
		stopOrder: stopOrder,
	}
}

func (p *shutdownPlugin) Info() plugin.PluginInfo                             { return p.info }
func (p *shutdownPlugin) Init(_ context.Context, _ plugin.Dependencies) error { return nil }
func (p *shutdownPlugin) Start(_ context.Context) error                       { return nil }

func (p *shutdownPlugin) Stop(ctx context.Context) error {
	// Record that we were called.
	if p.stopCount != nil {
		atomic.AddInt32(p.stopCount, 1)
	}

	// Simulate slow shutdown if configured.
	if p.stopDuration > 0 {
		select {
		case <-time.After(p.stopDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Record stop order.
	if p.stopOrder != nil {
		*p.stopOrder = append(*p.stopOrder, p.info.Name)
	}

	return p.stopErr
}

// testHTTPPlugin implements both Plugin and HTTPProvider.
type testHTTPPlugin struct {
	testPlugin
	routes []plugin.Route
}

func (p *testHTTPPlugin) Routes() []plugin.Route { return p.routes }

// testEventSubPlugin implements both Plugin and EventSubscriber.
type testEventSubPlugin struct {
	testPlugin
	subscriptions []plugin.Subscription
}

func (p *testEventSubPlugin) Subscriptions() []plugin.Subscription { return p.subscriptions }

// testBus records Subscribe calls for verification.
type testBus struct {
	subscriptions []struct{ topic string }
}

func (b *testBus) Publish(_ context.Context, _ plugin.Event) error { return nil }
func (b *testBus) Subscribe(topic string, _ plugin.EventHandler) (unsubscribe func()) {
	b.subscriptions = append(b.subscriptions, struct{ topic string }{topic})
	return func() {}
}
func (b *testBus) PublishAsync(_ context.Context, _ plugin.Event) {}
func (b *testBus) SubscribeAll(_ plugin.EventHandler) (unsubscribe func()) {
	return func() {}
}

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func testDeps() func(string) plugin.Dependencies {
	return func(name string) plugin.Dependencies {
		return plugin.Dependencies{
			Logger: testLogger().Named(name),
		}
	}
}

func TestRegister(t *testing.T) {
	reg := New(testLogger())

	p := newTestPlugin("alpha")
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Duplicate registration should fail.
	if err := reg.Register(p); err == nil {
		t.Fatal("Register() expected error for duplicate, got nil")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	reg := New(testLogger())
	p := &testPlugin{info: plugin.PluginInfo{Name: ""}}
	if err := reg.Register(p); err == nil {
		t.Fatal("Register() expected error for empty name, got nil")
	}
}

func TestValidateNoDeps(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Register(newTestPlugin("b"))

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d plugins, want 2", len(all))
	}
}

func TestValidateWithDeps(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("b", "a")) // b depends on a
	reg.Register(newTestPlugin("a"))

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// a should come before b in order.
	all := reg.All()
	aIdx, bIdx := -1, -1
	for i, p := range all {
		switch p.Info().Name {
		case "a":
			aIdx = i
		case "b":
			bIdx = i
		}
	}
	if aIdx >= bIdx {
		t.Errorf("expected a (idx %d) before b (idx %d)", aIdx, bIdx)
	}
}

func TestValidateCycleDetection(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a", "b"))
	reg.Register(newTestPlugin("b", "a"))

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected cycle error, got nil")
	}
}

func TestValidateMissingRequiredDep(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a", "missing")
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing required dep, got nil")
	}
}

func TestValidateDisablesOptionalWithMissingDep(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a", "missing")) // optional, dep doesn't exist

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !reg.IsDisabled("a") {
		t.Error("expected plugin 'a' to be disabled")
	}
}

func TestAPIVersionTooOld(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("old")
	p.info.APIVersion = 0 // below APIVersionMin
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for old API version, got nil")
	}
}

func TestAPIVersionTooNew(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("future")
	p.info.APIVersion = 999 // above APIVersionCurrent
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for future API version, got nil")
	}
}

func TestInitAll(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Register(newTestPlugin("b"))
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err != nil {
		t.Fatalf("InitAll() error = %v", err)
	}
}

func TestInitAllRequiredFails(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a")
	p.info.Required = true
	p.initErr = errors.New("init failed")
	reg.Register(p)
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err == nil {
		t.Fatal("InitAll() expected error for required plugin failure, got nil")
	}
}

func TestInitAllOptionalDisabledOnFailure(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a")
	p.initErr = errors.New("init failed")
	reg.Register(p)
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err != nil {
		t.Fatalf("InitAll() error = %v", err)
	}
	if !reg.IsDisabled("a") {
		t.Error("expected optional plugin 'a' to be disabled after init failure")
	}
}

func TestStartAllStopAll(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	if err := reg.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	reg.StopAll(ctx) // should not panic
}

func TestGet(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Validate()

	if _, ok := reg.Get("a"); !ok {
		t.Error("Get('a') returned false, want true")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("Get('nonexistent') returned true, want false")
	}
}

func TestAllRoutesHTTPProvider(t *testing.T) {
	reg := New(testLogger())

	hp := &testHTTPPlugin{
		testPlugin: *newTestPlugin("web"),
		routes: []plugin.Route{
			{Method: "GET", Path: "/test"},
		},
	}
	reg.Register(hp)
	reg.Register(newTestPlugin("noroutes")) // no HTTPProvider

	reg.Validate()
	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	routes := reg.AllRoutes()
	if len(routes) != 1 {
		t.Fatalf("AllRoutes() returned %d plugin route sets, want 1", len(routes))
	}
	if _, ok := routes["web"]; !ok {
		t.Error("AllRoutes() missing 'web' routes")
	}
}

func TestCascadeDisable(t *testing.T) {
	reg := New(testLogger())

	a := newTestPlugin("a")
	a.info.APIVersion = 0 // will be disabled (too old)

	b := newTestPlugin("b", "a") // depends on a

	reg.Register(a)
	reg.Register(b)

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !reg.IsDisabled("a") {
		t.Error("expected 'a' to be disabled (bad API version)")
	}
	if !reg.IsDisabled("b") {
		t.Error("expected 'b' to be cascade disabled")
	}
}

func TestInitAll_WiresEventSubscriber(t *testing.T) {
	reg := New(testLogger())

	var callCount int
	p := &testEventSubPlugin{
		testPlugin: *newTestPlugin("webhook"),
		subscriptions: []plugin.Subscription{
			{
				Topic: "recon.device.discovered",
				Handler: func(_ context.Context, _ plugin.Event) {
					callCount++
				},
			},
			{
				Topic: "recon.device.lost",
				Handler: func(_ context.Context, _ plugin.Event) {
					callCount++
				},
			},
		},
	}
	reg.Register(p)
	reg.Validate()

	bus := &testBus{}
	ctx := context.Background()
	err := reg.InitAll(ctx, func(name string) plugin.Dependencies {
		return plugin.Dependencies{
			Logger: testLogger().Named(name),
			Bus:    bus,
		}
	})
	if err != nil {
		t.Fatalf("InitAll() error = %v", err)
	}

	if len(bus.subscriptions) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(bus.subscriptions))
	}
	if bus.subscriptions[0].topic != "recon.device.discovered" {
		t.Errorf("subscription[0].topic = %q, want recon.device.discovered", bus.subscriptions[0].topic)
	}
	if bus.subscriptions[1].topic != "recon.device.lost" {
		t.Errorf("subscription[1].topic = %q, want recon.device.lost", bus.subscriptions[1].topic)
	}
}

// --- Graceful Shutdown Tests ---

func TestStopAll_ReverseOrder(t *testing.T) {
	// Test: SIGTERM triggers orderly shutdown (plugins stopped in reverse dependency order).
	var stopOrder []string
	reg := New(testLogger())

	// a has no deps, b depends on a, c depends on b
	// Start order: a, b, c
	// Stop order should be: c, b, a (reverse)
	a := newShutdownPlugin("a", &stopOrder)
	b := newShutdownPlugin("b", &stopOrder, "a")
	c := newShutdownPlugin("c", &stopOrder, "b")

	reg.Register(a)
	reg.Register(b)
	reg.Register(c)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)

	reg.StopAll(ctx)

	expected := []string{"c", "b", "a"}
	if len(stopOrder) != len(expected) {
		t.Fatalf("stop order length = %d, want %d", len(stopOrder), len(expected))
	}
	for i, name := range expected {
		if stopOrder[i] != name {
			t.Errorf("stop order[%d] = %q, want %q", i, stopOrder[i], name)
		}
	}
}

func TestStopAll_ReverseOrder_Table(t *testing.T) {
	tests := []struct {
		name       string
		plugins    []struct{ name string; deps []string }
		wantOrder  []string
	}{
		{
			name: "linear chain a->b->c",
			plugins: []struct{ name string; deps []string }{
				{"a", nil},
				{"b", []string{"a"}},
				{"c", []string{"b"}},
			},
			wantOrder: []string{"c", "b", "a"},
		},
		{
			name: "diamond a->b,c->d",
			plugins: []struct{ name string; deps []string }{
				{"a", nil},
				{"b", []string{"a"}},
				{"c", []string{"a"}},
				{"d", []string{"b", "c"}},
			},
			wantOrder: []string{"d", "c", "b", "a"}, // d first, then b/c (order may vary), then a last
		},
		{
			name: "independent plugins",
			plugins: []struct{ name string; deps []string }{
				{"x", nil},
				{"y", nil},
				{"z", nil},
			},
			wantOrder: nil, // any order is valid, just check all stopped
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stopOrder []string
			reg := New(testLogger())

			for _, p := range tc.plugins {
				reg.Register(newShutdownPlugin(p.name, &stopOrder, p.deps...))
			}
			reg.Validate()

			ctx := context.Background()
			reg.InitAll(ctx, testDeps())
			reg.StartAll(ctx)
			reg.StopAll(ctx)

			// All plugins should have stopped.
			if len(stopOrder) != len(tc.plugins) {
				t.Fatalf("stopped %d plugins, want %d", len(stopOrder), len(tc.plugins))
			}

			// For cases with specific expected order, verify it.
			if tc.wantOrder != nil {
				// For diamond case, just verify d is first and a is last.
				if tc.name == "diamond a->b,c->d" {
					if stopOrder[0] != "d" {
						t.Errorf("expected d to stop first, got %q", stopOrder[0])
					}
					if stopOrder[len(stopOrder)-1] != "a" {
						t.Errorf("expected a to stop last, got %q", stopOrder[len(stopOrder)-1])
					}
				} else {
					for i, name := range tc.wantOrder {
						if stopOrder[i] != name {
							t.Errorf("stop order[%d] = %q, want %q", i, stopOrder[i], name)
						}
					}
				}
			}
		})
	}
}

func TestStopAll_ErrorDoesNotBlockOthers(t *testing.T) {
	// Test: Plugin Stop() errors are logged but don't prevent other plugins from stopping.
	var stopOrder []string
	reg := New(testLogger())

	a := newShutdownPlugin("a", &stopOrder)
	b := newShutdownPlugin("b", &stopOrder, "a")
	b.stopErr = errors.New("b failed to stop")
	c := newShutdownPlugin("c", &stopOrder, "b")

	reg.Register(a)
	reg.Register(b)
	reg.Register(c)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)
	reg.StopAll(ctx)

	// All plugins should have had Stop() called despite b's error.
	if len(stopOrder) != 3 {
		t.Fatalf("stopped %d plugins, want 3 (all should stop despite errors)", len(stopOrder))
	}

	// Verify order is still correct (reverse dependency).
	expected := []string{"c", "b", "a"}
	for i, name := range expected {
		if stopOrder[i] != name {
			t.Errorf("stop order[%d] = %q, want %q", i, stopOrder[i], name)
		}
	}
}

func TestStopAll_MultipleErrorsAllStopped(t *testing.T) {
	// Multiple plugins fail but all are still called.
	var stopCount int32
	reg := New(testLogger())

	for i := 0; i < 5; i++ {
		p := newShutdownPlugin("p"+string(rune('a'+i)), nil)
		p.stopCount = &stopCount
		p.stopErr = errors.New("stop failed")
		reg.Register(p)
	}
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)
	reg.StopAll(ctx)

	if stopCount != 5 {
		t.Errorf("stop count = %d, want 5 (all plugins should have Stop() called)", stopCount)
	}
}

func TestStopAll_ContextTimeout(t *testing.T) {
	// Test: Per-plugin Stop() timeout enforced (slow plugin respects context deadline).
	var stopOrder []string
	reg := New(testLogger())

	fast := newShutdownPlugin("fast", &stopOrder)
	slow := newShutdownPlugin("slow", &stopOrder)
	slow.stopDuration = 5 * time.Second // Would take 5s without timeout

	reg.Register(fast)
	reg.Register(slow)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)

	// Use a short timeout - the slow plugin should respect ctx.Done().
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	reg.StopAll(shutdownCtx)
	elapsed := time.Since(start)

	// Should complete quickly due to context timeout, not wait for 5s.
	if elapsed > 500*time.Millisecond {
		t.Errorf("StopAll took %v, expected < 500ms with context timeout", elapsed)
	}

	// Fast plugin should have stopped successfully.
	// Slow plugin may or may not be in stopOrder depending on timing.
	foundFast := false
	for _, name := range stopOrder {
		if name == "fast" {
			foundFast = true
		}
	}
	if !foundFast {
		t.Error("expected 'fast' plugin to complete stop")
	}
}

func TestStopAll_CompletesWithinTimeout(t *testing.T) {
	// Test: Shutdown completes within configured maximum timeout.
	var stopOrder []string
	reg := New(testLogger())

	// Create several plugins with small delays.
	for i := 0; i < 3; i++ {
		p := newShutdownPlugin("p"+string(rune('a'+i)), &stopOrder)
		p.stopDuration = 10 * time.Millisecond
		reg.Register(p)
	}
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)

	// Use a generous timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	reg.StopAll(shutdownCtx)
	elapsed := time.Since(start)

	// Should complete well within timeout.
	if elapsed > 500*time.Millisecond {
		t.Errorf("StopAll took %v, expected < 500ms", elapsed)
	}

	// All plugins should have stopped.
	if len(stopOrder) != 3 {
		t.Errorf("stopped %d plugins, want 3", len(stopOrder))
	}
}

func TestStopAll_DisabledPluginsSkipped(t *testing.T) {
	// Disabled plugins should not have Stop() called.
	var stopCount int32
	reg := New(testLogger())

	active := newShutdownPlugin("active", nil)
	active.stopCount = &stopCount

	disabled := newShutdownPlugin("disabled", nil)
	disabled.stopCount = &stopCount
	disabled.info.APIVersion = 0 // Will be disabled due to old API version

	reg.Register(active)
	reg.Register(disabled)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)
	reg.StopAll(ctx)

	// Only active plugin should have Stop() called.
	if stopCount != 1 {
		t.Errorf("stop count = %d, want 1 (disabled plugin should be skipped)", stopCount)
	}
}

// --- Panic Recovery Tests ---

// panicPlugin is a test plugin that panics on configurable lifecycle methods.
type panicPlugin struct {
	testPlugin
	panicOnInit  bool
	panicOnStart bool
	panicOnStop  bool
}

func (p *panicPlugin) Init(ctx context.Context, deps plugin.Dependencies) error {
	if p.panicOnInit {
		panic("test panic in Init")
	}
	return p.testPlugin.Init(ctx, deps)
}

func (p *panicPlugin) Start(ctx context.Context) error {
	if p.panicOnStart {
		panic("test panic in Start")
	}
	return p.testPlugin.Start(ctx)
}

func (p *panicPlugin) Stop(ctx context.Context) error {
	if p.panicOnStop {
		panic("test panic in Stop")
	}
	return p.testPlugin.Stop(ctx)
}

func TestInitAll_PanicRecovery_OptionalPlugin(t *testing.T) {
	reg := New(testLogger())

	pp := &panicPlugin{
		testPlugin:  *newTestPlugin("panicker"),
		panicOnInit: true,
	}
	normal := newTestPlugin("normal")

	reg.Register(pp)
	reg.Register(normal)
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err != nil {
		t.Fatalf("InitAll() error = %v, want nil (optional panic should not propagate)", err)
	}

	if !reg.IsDisabled("panicker") {
		t.Error("expected panicking optional plugin to be disabled")
	}
	if reg.IsDisabled("normal") {
		t.Error("expected normal plugin to remain active")
	}
}

func TestInitAll_PanicRecovery_RequiredPlugin(t *testing.T) {
	reg := New(testLogger())

	pp := &panicPlugin{
		testPlugin:  *newTestPlugin("panicker"),
		panicOnInit: true,
	}
	pp.info.Required = true

	reg.Register(pp)
	reg.Validate()

	ctx := context.Background()
	err := reg.InitAll(ctx, testDeps())
	if err == nil {
		t.Fatal("InitAll() expected error for required panicking plugin, got nil")
	}

	if got := err.Error(); !strings.Contains(got, "panicked") {
		t.Errorf("error = %q, want it to contain 'panicked'", got)
	}
}

func TestStartAll_PanicRecovery_OptionalPlugin(t *testing.T) {
	reg := New(testLogger())

	pp := &panicPlugin{
		testPlugin:   *newTestPlugin("panicker"),
		panicOnStart: true,
	}
	normal := newTestPlugin("normal")

	reg.Register(pp)
	reg.Register(normal)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	if err := reg.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v, want nil (optional panic should not propagate)", err)
	}

	if !reg.IsDisabled("panicker") {
		t.Error("expected panicking optional plugin to be disabled")
	}
	if reg.IsDisabled("normal") {
		t.Error("expected normal plugin to remain active")
	}
}

func TestStartAll_PanicRecovery_RequiredPlugin(t *testing.T) {
	reg := New(testLogger())

	pp := &panicPlugin{
		testPlugin:   *newTestPlugin("panicker"),
		panicOnStart: true,
	}
	pp.info.Required = true

	reg.Register(pp)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	err := reg.StartAll(ctx)
	if err == nil {
		t.Fatal("StartAll() expected error for required panicking plugin, got nil")
	}

	if got := err.Error(); !strings.Contains(got, "panicked") {
		t.Errorf("error = %q, want it to contain 'panicked'", got)
	}
}

func TestStopAll_PanicRecovery(t *testing.T) {
	reg := New(testLogger())

	pp := &panicPlugin{
		testPlugin:  *newTestPlugin("panicker"),
		panicOnStop: true,
	}

	var stopOrder []string
	normal := newShutdownPlugin("normal", &stopOrder)

	reg.Register(pp)
	reg.Register(normal)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)

	// Should not panic -- recovery catches it.
	reg.StopAll(ctx)

	// The non-panicking plugin should still have had Stop() called.
	foundNormal := false
	for _, name := range stopOrder {
		if name == "normal" {
			foundNormal = true
		}
	}
	if !foundNormal {
		t.Error("expected normal plugin Stop() to be called despite other plugin panicking")
	}
}

func TestStopAll_ConcurrentSafety(t *testing.T) {
	// Verify StopAll is safe to call concurrently (uses RLock).
	var stopCount int32
	reg := New(testLogger())

	p := newShutdownPlugin("concurrent", nil)
	p.stopCount = &stopCount
	p.stopDuration = 50 * time.Millisecond

	reg.Register(p)
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())
	reg.StartAll(ctx)

	// Call StopAll from multiple goroutines.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.StopAll(ctx)
		}()
	}
	wg.Wait()

	// Stop should have been called 3 times (once per StopAll call).
	if stopCount != 3 {
		t.Errorf("stop count = %d, want 3", stopCount)
	}
}
