package autodoc

import (
	"fmt"
	"strings"
	"time"
)

// GenerateMarkdown produces a markdown summary of changelog entries grouped by date.
func GenerateMarkdown(entries []ChangelogEntry) string {
	if len(entries) == 0 {
		return "# Infrastructure Changelog\n\nNo changes recorded.\n"
	}

	var b strings.Builder
	b.WriteString("# Infrastructure Changelog\n\n")

	// Group entries by date (YYYY-MM-DD).
	groups := groupByDate(entries)

	for _, group := range groups {
		b.WriteString(fmt.Sprintf("## %s\n\n", group.date))

		for i := range group.entries {
			entry := &group.entries[i]
			ts := entry.CreatedAt.Format("15:04:05")
			icon := eventIcon(entry.EventType)
			b.WriteString(fmt.Sprintf("- %s **[%s]** %s %s\n", icon, ts, entry.Summary, sourceTag(entry.SourceModule)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// dateGroup holds entries for a single date.
type dateGroup struct {
	date    string
	entries []ChangelogEntry
}

// groupByDate groups entries by their date in YYYY-MM-DD format.
// Input entries should be in chronological order (oldest first).
// Output groups are in reverse chronological order (newest first).
func groupByDate(entries []ChangelogEntry) []dateGroup {
	groupMap := make(map[string][]ChangelogEntry)
	var dateOrder []string

	for i := range entries {
		date := entries[i].CreatedAt.Format("2006-01-02")
		if _, exists := groupMap[date]; !exists {
			dateOrder = append(dateOrder, date)
		}
		groupMap[date] = append(groupMap[date], entries[i])
	}

	// Reverse order so newest dates come first.
	groups := make([]dateGroup, len(dateOrder))
	for i, date := range dateOrder {
		groups[len(dateOrder)-1-i] = dateGroup{
			date:    date,
			entries: groupMap[date],
		}
	}
	return groups
}

// eventIcon returns a text label for the event type.
func eventIcon(eventType string) string {
	switch eventType {
	case TopicDeviceDiscovered:
		return "[NEW]"
	case TopicDeviceUpdated:
		return "[UPD]"
	case TopicDeviceLost:
		return "[LOST]"
	case TopicScanCompleted:
		return "[SCAN]"
	case TopicAlertTriggered:
		return "[ALERT]"
	case TopicAlertResolved:
		return "[OK]"
	default:
		return "[EVENT]"
	}
}

// sourceTag returns a formatted source module label.
func sourceTag(source string) string {
	if source == "" {
		return ""
	}
	return fmt.Sprintf("_(via %s)_", source)
}

// ParseDuration parses a human-friendly duration string like "7d", "30d", "24h".
// Falls back to the given default if parsing fails.
func ParseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}

	// Handle "Nd" format (days).
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	// Try standard Go duration parsing.
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
