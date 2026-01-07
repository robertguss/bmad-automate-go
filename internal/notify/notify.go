package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Notifier handles desktop notifications
type Notifier struct {
	enabled bool
}

// New creates a new notifier
func New(enabled bool) *Notifier {
	return &Notifier{enabled: enabled}
}

// SetEnabled enables or disables notifications
func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}

// IsEnabled returns whether notifications are enabled
func (n *Notifier) IsEnabled() bool {
	return n.enabled
}

// Notify sends a desktop notification
func (n *Notifier) Notify(title, message string) error {
	if !n.enabled {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return n.notifyMacOS(title, message)
	case "linux":
		return n.notifyLinux(title, message)
	default:
		// Notifications not supported on this platform
		return nil
	}
}

// NotifySuccess sends a success notification
func (n *Notifier) NotifySuccess(title, message string) error {
	return n.Notify(title, message)
}

// NotifyError sends an error notification
func (n *Notifier) NotifyError(title, message string) error {
	return n.Notify(title, message)
}

// NotifyQueueComplete sends notification when queue completes
func (n *Notifier) NotifyQueueComplete(total, succeeded, failed int) error {
	var title string
	var message string

	if failed == 0 {
		title = "Queue Complete"
		message = fmt.Sprintf("All %d stories completed successfully", total)
	} else {
		title = "Queue Complete with Errors"
		message = fmt.Sprintf("%d succeeded, %d failed out of %d total", succeeded, failed, total)
	}

	return n.Notify(title, message)
}

// NotifyStoryComplete sends notification when a story completes
func (n *Notifier) NotifyStoryComplete(storyKey string, success bool) error {
	var title, message string

	if success {
		title = "Story Complete"
		message = fmt.Sprintf("%s completed successfully", storyKey)
	} else {
		title = "Story Failed"
		message = fmt.Sprintf("%s failed during execution", storyKey)
	}

	return n.Notify(title, message)
}

// notifyMacOS sends notification using osascript on macOS
func (n *Notifier) notifyMacOS(title, message string) error {
	// Escape quotes in title and message
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// notifyLinux sends notification using notify-send on Linux
func (n *Notifier) notifyLinux(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}
