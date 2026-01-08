// Package util provides shared utility functions used across the application.
// QUAL-002: Consolidates duplicated utility functions.
package util

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration for human-readable display.
// - Under 1 minute: "45s"
// - 1 minute or more: "5m 30s"
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %02ds", minutes, seconds)
}

// FormatDurationExtended formats a duration with hours support.
// - Under 1 minute: "45s"
// - Under 1 hour: "5m 30s"
// - 1 hour or more: "1h 23m"
func FormatDurationExtended(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %02ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

// FormatDurationCompact formats a duration in a compact format for statistics.
// - Under 1 second: "500ms"
// - Under 1 minute: "45.5s"
// - Under 1 hour: "5m30s"
// - 1 hour or more: "1h23m"
func FormatDurationCompact(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// FormatDurationLong formats a duration with more detail.
// - Under 1 minute: "45 seconds"
// - Under 1 hour: "5 minutes 30 seconds"
// - 1 hour or more: "1 hour 23 minutes"
func FormatDurationLong(d time.Duration) string {
	if d < time.Minute {
		secs := int(d.Seconds())
		if secs == 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", secs)
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		minUnit := "minutes"
		if mins == 1 {
			minUnit = "minute"
		}
		if secs == 0 {
			return fmt.Sprintf("%d %s", mins, minUnit)
		}
		secUnit := "seconds"
		if secs == 1 {
			secUnit = "second"
		}
		return fmt.Sprintf("%d %s %d %s", mins, minUnit, secs, secUnit)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	hourUnit := "hours"
	if hours == 1 {
		hourUnit = "hour"
	}
	minUnit := "minutes"
	if mins == 1 {
		minUnit = "minute"
	}
	if mins == 0 {
		return fmt.Sprintf("%d %s", hours, hourUnit)
	}
	return fmt.Sprintf("%d %s %d %s", hours, hourUnit, mins, minUnit)
}
