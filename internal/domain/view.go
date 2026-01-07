package domain

// View represents the current active view
type View int

const (
	ViewDashboard View = iota
	ViewStoryList
	ViewQueue
	ViewExecution
	ViewTimeline
	ViewDiff
	ViewHistory
	ViewStats
	ViewSettings
)

// String returns the display name of the view
func (v View) String() string {
	switch v {
	case ViewDashboard:
		return "Dashboard"
	case ViewStoryList:
		return "Stories"
	case ViewQueue:
		return "Queue"
	case ViewExecution:
		return "Execution"
	case ViewTimeline:
		return "Timeline"
	case ViewDiff:
		return "Diff"
	case ViewHistory:
		return "History"
	case ViewStats:
		return "Statistics"
	case ViewSettings:
		return "Settings"
	default:
		return "Unknown"
	}
}

// Shortcut returns the keyboard shortcut for the view
func (v View) Shortcut() string {
	switch v {
	case ViewDashboard:
		return "d"
	case ViewStoryList:
		return "s"
	case ViewQueue:
		return "q"
	case ViewHistory:
		return "h"
	case ViewStats:
		return "a"
	case ViewSettings:
		return "o"
	default:
		return ""
	}
}
