package docs

import (
	"strings"
	"testing"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		new         string
		wantAdds    int
		wantRemoves int
		wantContext int
	}{
		{
			name:        "identical content",
			old:         "line1\nline2\nline3",
			new:         "line1\nline2\nline3",
			wantAdds:    0,
			wantRemoves: 0,
			wantContext: 3,
		},
		{
			name:        "additions only",
			old:         "line1\nline3",
			new:         "line1\nline2\nline3",
			wantAdds:    1,
			wantRemoves: 0,
			wantContext: 2,
		},
		{
			name:        "deletions only",
			old:         "line1\nline2\nline3",
			new:         "line1\nline3",
			wantAdds:    0,
			wantRemoves: 1,
			wantContext: 2,
		},
		{
			name:        "mixed changes",
			old:         "aaa\nbbb\nccc\nddd",
			new:         "aaa\nBBB\nccc\neee",
			wantAdds:    2,
			wantRemoves: 2,
			wantContext: 2,
		},
		{
			name:        "empty old",
			old:         "",
			new:         "line1\nline2",
			wantAdds:    2,
			wantRemoves: 0,
			wantContext: 0,
		},
		{
			name:        "empty new",
			old:         "line1\nline2",
			new:         "",
			wantAdds:    0,
			wantRemoves: 2,
			wantContext: 0,
		},
		{
			name:        "both empty",
			old:         "",
			new:         "",
			wantAdds:    0,
			wantRemoves: 0,
			wantContext: 0,
		},
		{
			name:        "completely different",
			old:         "aaa\nbbb",
			new:         "ccc\nddd",
			wantAdds:    2,
			wantRemoves: 2,
			wantContext: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := ComputeDiff(tt.old, tt.new)

			adds, removes, context := 0, 0, 0
			for _, l := range lines {
				switch l.Type {
				case DiffAdd:
					adds++
				case DiffRemove:
					removes++
				case DiffContext:
					context++
				}
			}

			if adds != tt.wantAdds {
				t.Errorf("adds = %d, want %d", adds, tt.wantAdds)
			}
			if removes != tt.wantRemoves {
				t.Errorf("removes = %d, want %d", removes, tt.wantRemoves)
			}
			if context != tt.wantContext {
				t.Errorf("context = %d, want %d", context, tt.wantContext)
			}
		})
	}
}

func TestComputeDiff_LineNumbers(t *testing.T) {
	old := "aaa\nbbb\nccc"
	updated := "aaa\nBBB\nccc"

	lines := ComputeDiff(old, updated)

	// Expect: context(aaa), remove(bbb), add(BBB), context(ccc)
	if len(lines) != 4 {
		t.Fatalf("len(lines) = %d, want 4", len(lines))
	}

	// Context line "aaa": old=1, new=1
	if lines[0].OldLineNo != 1 || lines[0].NewLineNo != 1 {
		t.Errorf("line 0: old=%d new=%d, want old=1 new=1", lines[0].OldLineNo, lines[0].NewLineNo)
	}

	// Remove line "bbb": old=2
	if lines[1].Type != DiffRemove || lines[1].OldLineNo != 2 {
		t.Errorf("line 1: type=%d old=%d, want type=DiffRemove old=2", lines[1].Type, lines[1].OldLineNo)
	}

	// Add line "BBB": new=2
	if lines[2].Type != DiffAdd || lines[2].NewLineNo != 2 {
		t.Errorf("line 2: type=%d new=%d, want type=DiffAdd new=2", lines[2].Type, lines[2].NewLineNo)
	}

	// Context line "ccc": old=3, new=3
	if lines[3].OldLineNo != 3 || lines[3].NewLineNo != 3 {
		t.Errorf("line 3: old=%d new=%d, want old=3 new=3", lines[3].OldLineNo, lines[3].NewLineNo)
	}
}

func TestFormatUnifiedDiff(t *testing.T) {
	tests := []struct {
		name         string
		old          string
		new          string
		contextLines int
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "identical returns empty",
			old:          "line1\nline2\nline3",
			new:          "line1\nline2\nline3",
			contextLines: 3,
			wantEmpty:    true,
		},
		{
			name:         "addition shows plus prefix",
			old:          "aaa\nccc",
			new:          "aaa\nbbb\nccc",
			contextLines: 1,
			wantContains: []string{"+bbb", "@@"},
		},
		{
			name:         "removal shows minus prefix",
			old:          "aaa\nbbb\nccc",
			new:          "aaa\nccc",
			contextLines: 1,
			wantContains: []string{"-bbb", "@@"},
		},
		{
			name:         "both empty returns empty",
			old:          "",
			new:          "",
			contextLines: 3,
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := ComputeDiff(tt.old, tt.new)
			result := FormatUnifiedDiff(lines, tt.contextLines)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty result, got:\n%s", result)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result does not contain %q:\n%s", want, result)
				}
			}
		})
	}
}

func TestComputeLCS(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want int
	}{
		{"both empty", nil, nil, 0},
		{"one empty", []string{"a"}, nil, 0},
		{"identical", []string{"a", "b", "c"}, []string{"a", "b", "c"}, 3},
		{"no common", []string{"a", "b"}, []string{"c", "d"}, 0},
		{"partial overlap", []string{"a", "b", "c", "d"}, []string{"a", "c", "d", "e"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeLCS(tt.a, tt.b)
			if len(result) != tt.want {
				t.Errorf("len(computeLCS) = %d, want %d", len(result), tt.want)
			}
		})
	}
}
