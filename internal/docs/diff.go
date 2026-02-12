package docs

import (
	"fmt"
	"strings"
)

// DiffLineType indicates whether a diff line is context, added, or removed.
type DiffLineType int

const (
	// DiffContext indicates an unchanged line.
	DiffContext DiffLineType = iota
	// DiffAdd indicates a line present only in the new content.
	DiffAdd
	// DiffRemove indicates a line present only in the old content.
	DiffRemove
)

// DiffLine represents a single line in a unified diff.
type DiffLine struct {
	Type      DiffLineType `json:"type"`
	Content   string       `json:"content"`
	OldLineNo int          `json:"old_line_no,omitempty"`
	NewLineNo int          `json:"new_line_no,omitempty"`
}

// ComputeDiff computes a line-based diff between old and new content using LCS.
func ComputeDiff(old, updated string) []DiffLine {
	oldLines := splitLines(old)
	newLines := splitLines(updated)

	lcs := computeLCS(oldLines, newLines)

	var result []DiffLine
	oi, ni, li := 0, 0, 0

	for li < len(lcs) {
		// Emit removals from old until we reach the next LCS line.
		for oi < len(oldLines) && oldLines[oi] != lcs[li] {
			oi++
			result = append(result, DiffLine{
				Type:      DiffRemove,
				Content:   oldLines[oi-1],
				OldLineNo: oi,
			})
		}
		// Emit additions from new until we reach the next LCS line.
		for ni < len(newLines) && newLines[ni] != lcs[li] {
			ni++
			result = append(result, DiffLine{
				Type:      DiffAdd,
				Content:   newLines[ni-1],
				NewLineNo: ni,
			})
		}
		// Emit the common (context) line.
		oi++
		ni++
		li++
		result = append(result, DiffLine{
			Type:      DiffContext,
			Content:   oldLines[oi-1],
			OldLineNo: oi,
			NewLineNo: ni,
		})
	}

	// Remaining old lines are removals.
	for oi < len(oldLines) {
		oi++
		result = append(result, DiffLine{
			Type:      DiffRemove,
			Content:   oldLines[oi-1],
			OldLineNo: oi,
		})
	}

	// Remaining new lines are additions.
	for ni < len(newLines) {
		ni++
		result = append(result, DiffLine{
			Type:      DiffAdd,
			Content:   newLines[ni-1],
			NewLineNo: ni,
		})
	}

	return result
}

// FormatUnifiedDiff formats diff lines as unified diff text with the given
// number of context lines around each change.
func FormatUnifiedDiff(lines []DiffLine, contextLines int) string {
	if len(lines) == 0 {
		return ""
	}

	// Find runs of changes and include context around them.
	var b strings.Builder

	// Identify which lines to include (changed lines + surrounding context).
	include := make([]bool, len(lines))
	for i, line := range lines {
		if line.Type == DiffContext {
			continue
		}
		lo := i - contextLines
		if lo < 0 {
			lo = 0
		}
		hi := i + contextLines
		if hi >= len(lines) {
			hi = len(lines) - 1
		}
		for j := lo; j <= hi; j++ {
			include[j] = true
		}
	}

	// If nothing changed, return empty string.
	anyChange := false
	for _, line := range lines {
		if line.Type != DiffContext {
			anyChange = true
			break
		}
	}
	if !anyChange {
		return ""
	}

	inHunk := false
	for i, line := range lines {
		if !include[i] {
			if inHunk {
				inHunk = false
			}
			continue
		}

		if !inHunk {
			// Start a new hunk header.
			oldStart, newStart, oldCount, newCount := hunkRange(lines, include, i, contextLines)
			fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
			inHunk = true
		}

		switch line.Type {
		case DiffContext:
			fmt.Fprintf(&b, " %s\n", line.Content)
		case DiffAdd:
			fmt.Fprintf(&b, "+%s\n", line.Content)
		case DiffRemove:
			fmt.Fprintf(&b, "-%s\n", line.Content)
		}
	}

	return b.String()
}

// hunkRange computes the old and new starting lines and counts for a hunk
// beginning at position start in the lines slice.
func hunkRange(lines []DiffLine, include []bool, start, contextLines int) (oldStart, newStart, oldCount, newCount int) {
	_ = contextLines // used implicitly through the include mask

	oldStart = 0
	newStart = 0
	firstSet := false

	for i := start; i < len(lines); i++ {
		if !include[i] {
			break
		}
		if !firstSet {
			switch lines[i].Type {
			case DiffContext:
				oldStart = lines[i].OldLineNo
				newStart = lines[i].NewLineNo
			case DiffRemove:
				oldStart = lines[i].OldLineNo
				// Find the first new line number from a following add or context.
				newStart = findNewStart(lines, i)
			case DiffAdd:
				newStart = lines[i].NewLineNo
				oldStart = findOldStart(lines, i)
			}
			firstSet = true
		}

		switch lines[i].Type {
		case DiffContext:
			oldCount++
			newCount++
		case DiffRemove:
			oldCount++
		case DiffAdd:
			newCount++
		}
	}

	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}

	return oldStart, newStart, oldCount, newCount
}

// findNewStart scans forward from pos to find the first NewLineNo.
func findNewStart(lines []DiffLine, pos int) int {
	for i := pos; i < len(lines); i++ {
		if lines[i].NewLineNo > 0 {
			return lines[i].NewLineNo
		}
	}
	return 1
}

// findOldStart scans forward from pos to find the first OldLineNo.
func findOldStart(lines []DiffLine, pos int) int {
	for i := pos; i < len(lines); i++ {
		if lines[i].OldLineNo > 0 {
			return lines[i].OldLineNo
		}
	}
	return 1
}

// splitLines splits content into lines. An empty string returns an empty slice.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// computeLCS computes the longest common subsequence of two string slices.
func computeLCS(a, b []string) []string {
	m := len(a)
	n := len(b)

	// Build the LCS table.
	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			switch {
			case a[i-1] == b[j-1]:
				table[i][j] = table[i-1][j-1] + 1
			case table[i-1][j] >= table[i][j-1]:
				table[i][j] = table[i-1][j]
			default:
				table[i][j] = table[i][j-1]
			}
		}
	}

	// Backtrack to find the LCS.
	lcsLen := table[m][n]
	if lcsLen == 0 {
		return nil
	}

	result := make([]string, lcsLen)
	i, j := m, n
	idx := lcsLen - 1
	for i > 0 && j > 0 {
		switch {
		case a[i-1] == b[j-1]:
			result[idx] = a[i-1]
			idx--
			i--
			j--
		case table[i-1][j] >= table[i][j-1]:
			i--
		default:
			j--
		}
	}

	return result
}
