package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

// newTestModule creates a Module wired to an in-memory store for handler tests.
func newTestModule(t *testing.T) *Module {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "docs", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return &Module{
		logger: zap.NewNop(),
		store:  NewStore(db.DB()),
		cfg:    DefaultConfig(),
	}
}

// -- handleListApplications tests --

func TestHandleListApplications_Empty(t *testing.T) {
	m := newTestModule(t)
	req := httptest.NewRequest(http.MethodGet, "/applications", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListApplications(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var items []Application
	if err := json.Unmarshal(resp["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}

	var total int
	if err := json.Unmarshal(resp["total"], &total); err != nil {
		t.Fatalf("decode total: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestHandleListApplications_WithData(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Proxmox", AppType: "hypervisor",
		Collector: "docker", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/applications", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListApplications(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var items []Application
	if err := json.Unmarshal(resp["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "app-001" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "app-001")
	}
	if items[0].Name != "Proxmox" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "Proxmox")
	}
}

func TestHandleListApplications_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/applications", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListApplications(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleListApplications_WithFilters(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	apps := []*Application{
		{
			ID: "app-001", Name: "Proxmox", AppType: "hypervisor",
			Collector: "docker", Status: "active", Metadata: "{}",
			DiscoveredAt: now, UpdatedAt: now,
		},
		{
			ID: "app-002", Name: "Nginx", AppType: "web_server",
			Collector: "docker", Status: "active", Metadata: "{}",
			DiscoveredAt: now, UpdatedAt: now.Add(time.Second),
		},
	}
	for _, a := range apps {
		if err := m.store.InsertApplication(context.Background(), a); err != nil {
			t.Fatalf("insert application: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/applications?type=hypervisor", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListApplications(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var items []Application
	if err := json.Unmarshal(resp["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].AppType != "hypervisor" {
		t.Errorf("items[0].AppType = %q, want %q", items[0].AppType, "hypervisor")
	}
}

// -- handleGetApplication tests --

func TestHandleGetApplication_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/applications/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleGetApplication(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetApplication_Found(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Proxmox", AppType: "hypervisor",
		DeviceID: "dev-001", Collector: "docker", Status: "active",
		Metadata: `{"version":"8.0"}`, DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/applications/app-001", http.NoBody)
	req.SetPathValue("id", "app-001")
	w := httptest.NewRecorder()

	m.handleGetApplication(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Application
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "app-001" {
		t.Errorf("ID = %q, want %q", got.ID, "app-001")
	}
	if got.Name != "Proxmox" {
		t.Errorf("Name = %q, want %q", got.Name, "Proxmox")
	}
	if got.AppType != "hypervisor" {
		t.Errorf("AppType = %q, want %q", got.AppType, "hypervisor")
	}
}

func TestHandleGetApplication_EmptyID(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/applications/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleGetApplication(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleGetApplication_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/applications/app-001", http.NoBody)
	req.SetPathValue("id", "app-001")
	w := httptest.NewRecorder()

	m.handleGetApplication(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleListSnapshots tests --

func TestHandleListSnapshots_Empty(t *testing.T) {
	m := newTestModule(t)
	req := httptest.NewRequest(http.MethodGet, "/snapshots", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListSnapshots(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var snapshots []Snapshot
	if err := json.NewDecoder(w.Body).Decode(&snapshots); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("len(snapshots) = %d, want 0", len(snapshots))
	}
}

func TestHandleListSnapshots_WithData(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "config data", Format: "json", SizeBytes: 11,
		Source: "manual", CapturedAt: now,
	}
	if err := m.store.InsertSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/snapshots?application_id=app-001", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListSnapshots(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var snapshots []Snapshot
	if err := json.NewDecoder(w.Body).Decode(&snapshots); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("len(snapshots) = %d, want 1", len(snapshots))
	}
	if snapshots[0].ID != "snap-001" {
		t.Errorf("snapshots[0].ID = %q, want %q", snapshots[0].ID, "snap-001")
	}
}

func TestHandleListSnapshots_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/snapshots", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListSnapshots(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleGetSnapshot tests --

func TestHandleGetSnapshot_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/snapshots/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleGetSnapshot(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetSnapshot_Found(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "abc123",
		Content: `{"key":"value"}`, Format: "json", SizeBytes: 15,
		Source: "manual", CapturedAt: now,
	}
	if err := m.store.InsertSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/snapshots/snap-001", http.NoBody)
	req.SetPathValue("id", "snap-001")
	w := httptest.NewRecorder()

	m.handleGetSnapshot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Snapshot
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "snap-001" {
		t.Errorf("ID = %q, want %q", got.ID, "snap-001")
	}
	if got.Content != `{"key":"value"}` {
		t.Errorf("Content = %q, want %q", got.Content, `{"key":"value"}`)
	}
}

func TestHandleGetSnapshot_EmptyID(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/snapshots/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleGetSnapshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleGetSnapshot_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/snapshots/snap-001", http.NoBody)
	req.SetPathValue("id", "snap-001")
	w := httptest.NewRecorder()

	m.handleGetSnapshot(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleCreateSnapshot tests --

func TestHandleCreateSnapshot_Success(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	body := `{"application_id":"app-001","content":"{\"key\":\"value\"}","format":"json"}`
	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got Snapshot
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Error("ID is empty, want generated UUID")
	}
	if got.ApplicationID != "app-001" {
		t.Errorf("ApplicationID = %q, want %q", got.ApplicationID, "app-001")
	}
	if got.ContentHash == "" {
		t.Error("ContentHash is empty, want SHA-256 hash")
	}
	if got.Content != `{"key":"value"}` {
		t.Errorf("Content = %q, want %q", got.Content, `{"key":"value"}`)
	}
	if got.Format != "json" {
		t.Errorf("Format = %q, want %q", got.Format, "json")
	}
	if got.SizeBytes != len(`{"key":"value"}`) {
		t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, len(`{"key":"value"}`))
	}
	if got.Source != "manual" {
		t.Errorf("Source = %q, want %q", got.Source, "manual")
	}
}

func TestHandleCreateSnapshot_DefaultFormat(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	// Omit the format field -- should default to "json".
	body := `{"application_id":"app-001","content":"some data"}`
	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var got Snapshot
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Format != "json" {
		t.Errorf("Format = %q, want default %q", got.Format, "json")
	}
}

func TestHandleCreateSnapshot_InvalidJSON(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSnapshot_MissingApplicationID(t *testing.T) {
	m := newTestModule(t)

	body := `{"content":"some data","format":"json"}`
	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSnapshot_MissingContent(t *testing.T) {
	m := newTestModule(t)

	body := `{"application_id":"app-001","format":"json"}`
	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSnapshot_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	body := `{"application_id":"app-001","content":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleCreateSnapshot(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleApplicationHistory tests --

func TestHandleApplicationHistory_WithData(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	for i := 0; i < 3; i++ {
		snap := &Snapshot{
			ID: fmt.Sprintf("snap-%03d", i), ApplicationID: "app-001",
			ContentHash: fmt.Sprintf("h%d", i), Content: fmt.Sprintf("cfg %d", i),
			Format: "text", SizeBytes: 5, Source: "manual",
			CapturedAt: now.Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.InsertSnapshot(context.Background(), snap); err != nil {
			t.Fatalf("insert snapshot %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/applications/app-001/history?limit=10", http.NoBody)
	req.SetPathValue("id", "app-001")
	w := httptest.NewRecorder()

	m.handleApplicationHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(resp["snapshots"], &snapshots); err != nil {
		t.Fatalf("decode snapshots: %v", err)
	}
	if len(snapshots) != 3 {
		t.Errorf("len(snapshots) = %d, want 3", len(snapshots))
	}

	var total int
	if err := json.Unmarshal(resp["total"], &total); err != nil {
		t.Fatalf("decode total: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestHandleApplicationHistory_Empty(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/applications/app-001/history", http.NoBody)
	req.SetPathValue("id", "app-001")
	w := httptest.NewRecorder()

	m.handleApplicationHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(resp["snapshots"], &snapshots); err != nil {
		t.Fatalf("decode snapshots: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("len(snapshots) = %d, want 0", len(snapshots))
	}
}

func TestHandleApplicationHistory_InvalidApp(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/applications/nonexistent/history", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleApplicationHistory(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- handleSnapshotDiff tests --

func TestHandleSnapshotDiff_Normal(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	snap1 := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "line1\nline2\nline3", Format: "text", SizeBytes: 17,
		Source: "manual", CapturedAt: now,
	}
	snap2 := &Snapshot{
		ID: "snap-002", ApplicationID: "app-001", ContentHash: "h2",
		Content: "line1\nLINE2\nline3", Format: "text", SizeBytes: 17,
		Source: "manual", CapturedAt: now.Add(time.Minute),
	}
	if err := m.store.InsertSnapshot(context.Background(), snap1); err != nil {
		t.Fatalf("insert snap1: %v", err)
	}
	if err := m.store.InsertSnapshot(context.Background(), snap2); err != nil {
		t.Fatalf("insert snap2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/snapshots/snap-001/diff/snap-002", http.NoBody)
	req.SetPathValue("id", "snap-001")
	req.SetPathValue("other_id", "snap-002")
	w := httptest.NewRecorder()

	m.handleSnapshotDiff(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp DiffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.OldSnapshotID != "snap-001" {
		t.Errorf("OldSnapshotID = %q, want %q", resp.OldSnapshotID, "snap-001")
	}
	if resp.NewSnapshotID != "snap-002" {
		t.Errorf("NewSnapshotID = %q, want %q", resp.NewSnapshotID, "snap-002")
	}
	if resp.DiffText == "" {
		t.Error("DiffText is empty, expected diff output")
	}
}

func TestHandleSnapshotDiff_SameSnapshot(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "identical content", Format: "text", SizeBytes: 17,
		Source: "manual", CapturedAt: now,
	}
	if err := m.store.InsertSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/snapshots/snap-001/diff/snap-001", http.NoBody)
	req.SetPathValue("id", "snap-001")
	req.SetPathValue("other_id", "snap-001")
	w := httptest.NewRecorder()

	m.handleSnapshotDiff(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp DiffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Same content means empty diff.
	if resp.DiffText != "" {
		t.Errorf("DiffText = %q, want empty (identical content)", resp.DiffText)
	}
}

func TestHandleSnapshotDiff_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/snapshots/nonexistent/diff/also-nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	req.SetPathValue("other_id", "also-nonexistent")
	w := httptest.NewRecorder()

	m.handleSnapshotDiff(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- handleDeleteSnapshot tests --

func TestHandleDeleteSnapshot_Success(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(context.Background(), app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "to delete", Format: "text", SizeBytes: 9,
		Source: "manual", CapturedAt: now,
	}
	if err := m.store.InsertSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/snapshots/snap-001", http.NoBody)
	req.SetPathValue("id", "snap-001")
	w := httptest.NewRecorder()

	m.handleDeleteSnapshot(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify it's gone.
	got, err := m.store.GetSnapshot(context.Background(), "snap-001")
	if err != nil {
		t.Fatalf("get snapshot after delete: %v", err)
	}
	if got != nil {
		t.Error("snapshot should have been deleted")
	}
}

func TestHandleDeleteSnapshot_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodDelete, "/snapshots/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleDeleteSnapshot(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- docsQueryInt tests --

func TestDocsQueryInt(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int
		want       int
	}{
		{
			name:       "no param returns default",
			query:      "",
			key:        "limit",
			defaultVal: 50,
			want:       50,
		},
		{
			name:       "valid param",
			query:      "limit=25",
			key:        "limit",
			defaultVal: 50,
			want:       25,
		},
		{
			name:       "negative returns default",
			query:      "limit=-5",
			key:        "limit",
			defaultVal: 50,
			want:       50,
		},
		{
			name:       "non-numeric returns default",
			query:      "limit=abc",
			key:        "limit",
			defaultVal: 50,
			want:       50,
		},
		{
			name:       "zero returns zero",
			query:      "offset=0",
			key:        "offset",
			defaultVal: 10,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			got := docsQueryInt(req, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("docsQueryInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// -- handleListCollectors tests --

func TestHandleListCollectors_Empty(t *testing.T) {
	m := newTestModule(t)
	req := httptest.NewRequest(http.MethodGet, "/collectors", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListCollectors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var infos []CollectorInfo
	if err := json.NewDecoder(w.Body).Decode(&infos); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("len(infos) = %d, want 0", len(infos))
	}
}

func TestHandleListCollectors_WithCollectors(t *testing.T) {
	m := newTestModule(t)
	m.collectors = []Collector{
		&mockCollector{name: "docker", available: true},
		&mockCollector{name: "systemd", available: false},
	}

	req := httptest.NewRequest(http.MethodGet, "/collectors", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListCollectors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var infos []CollectorInfo
	if err := json.NewDecoder(w.Body).Decode(&infos); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("len(infos) = %d, want 2", len(infos))
	}
	if infos[0].Name != "docker" {
		t.Errorf("infos[0].Name = %q, want %q", infos[0].Name, "docker")
	}
	if !infos[0].Available {
		t.Error("infos[0].Available = false, want true")
	}
	if infos[1].Name != "systemd" {
		t.Errorf("infos[1].Name = %q, want %q", infos[1].Name, "systemd")
	}
	if infos[1].Available {
		t.Error("infos[1].Available = true, want false")
	}
}

// -- handleTriggerCollection tests --

func TestHandleTriggerCollection(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	m.collectors = []Collector{
		&mockCollector{
			name:      "test",
			available: true,
			apps: []Application{
				{
					ID: "app-001", Name: "nginx", AppType: "docker-container",
					Collector: "test", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			configs: map[string]*CollectedConfig{
				"app-001": {Content: `{"image":"nginx"}`, Format: "json"},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/collect", http.NoBody)
	w := httptest.NewRecorder()

	m.handleTriggerCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result CollectionResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.AppsDiscovered != 1 {
		t.Errorf("AppsDiscovered = %d, want 1", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 1 {
		t.Errorf("SnapshotsCreated = %d, want 1", result.SnapshotsCreated)
	}
}

func TestHandleTriggerCollection_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodPost, "/collect", http.NoBody)
	w := httptest.NewRecorder()

	m.handleTriggerCollection(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleTriggerCollectorByName tests --

func TestHandleTriggerCollectorByName(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	m.collectors = []Collector{
		&mockCollector{
			name:      "docker",
			available: true,
			apps: []Application{
				{
					ID: "app-001", Name: "nginx", AppType: "docker-container",
					Collector: "docker", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			configs: map[string]*CollectedConfig{
				"app-001": {Content: `{"from":"docker"}`, Format: "json"},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/collect/docker", http.NoBody)
	req.SetPathValue("collector", "docker")
	w := httptest.NewRecorder()

	m.handleTriggerCollectorByName(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result CollectionResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.AppsDiscovered != 1 {
		t.Errorf("AppsDiscovered = %d, want 1", result.AppsDiscovered)
	}
}

func TestHandleTriggerCollectorByName_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodPost, "/collect/nonexistent", http.NoBody)
	req.SetPathValue("collector", "nonexistent")
	w := httptest.NewRecorder()

	m.handleTriggerCollectorByName(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTriggerCollectorByName_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodPost, "/collect/docker", http.NoBody)
	req.SetPathValue("collector", "docker")
	w := httptest.NewRecorder()

	m.handleTriggerCollectorByName(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
