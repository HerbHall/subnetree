package catalog

import (
	"testing"
)

func TestCatalog_LoadsSuccessfully(t *testing.T) {
	cat := NewCatalog()
	entries, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 48 {
		t.Fatalf("expected 48 entries, got %d", len(entries))
	}
}

func TestCatalog_AllEntriesHaveRequiredFields(t *testing.T) {
	cat := NewCatalog()
	entries, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := range entries {
		e := &entries[i]
		if e.Name == "" {
			t.Errorf("entry %d: missing name", i)
		}
		if e.Description == "" {
			t.Errorf("entry %d (%s): missing description", i, e.Name)
		}
		if e.Category == "" {
			t.Errorf("entry %d (%s): missing category", i, e.Name)
		}
		if e.GitHubURL == "" {
			t.Errorf("entry %d (%s): missing github_url", i, e.Name)
		}
		if e.DockerImage == "" {
			t.Errorf("entry %d (%s): missing docker_image", i, e.Name)
		}
		if e.Stars <= 0 {
			t.Errorf("entry %d (%s): stars must be positive, got %d", i, e.Name, e.Stars)
		}
		if e.License == "" {
			t.Errorf("entry %d (%s): missing license", i, e.Name)
		}
		if e.Language == "" {
			t.Errorf("entry %d (%s): missing language", i, e.Name)
		}
		if e.MinRAMMB <= 0 {
			t.Errorf("entry %d (%s): min_ram_mb must be positive, got %d", i, e.Name, e.MinRAMMB)
		}
		if len(e.SupportedTiers) == 0 {
			t.Errorf("entry %d (%s): missing supported_tiers", i, e.Name)
		}
		if e.IntegrationStatus == "" {
			t.Errorf("entry %d (%s): missing integration_status", i, e.Name)
		}
	}
}

func TestCatalog_TierRangesValid(t *testing.T) {
	cat := NewCatalog()
	entries, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := range entries {
		e := &entries[i]
		for _, tier := range e.SupportedTiers {
			if tier < 0 || tier > 4 {
				t.Errorf("entry %d (%s): invalid tier %d (must be 0-4)", i, e.Name, tier)
			}
		}
	}
}

func TestCatalog_NoDuplicateNames(t *testing.T) {
	cat := NewCatalog()
	entries, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seen := make(map[string]bool, len(entries))
	for i := range entries {
		name := entries[i].Name
		if seen[name] {
			t.Errorf("duplicate entry name: %s", name)
		}
		seen[name] = true
	}
}

func TestCatalog_ReturnsCopy(t *testing.T) {
	cat := NewCatalog()
	entries1, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries2, err := cat.Entries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the first slice and verify the second is unaffected.
	original := entries1[0].Name
	entries1[0].Name = "MUTATED"
	if entries2[0].Name != original {
		t.Errorf("Entries() did not return a copy: mutation leaked")
	}
}
