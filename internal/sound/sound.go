package sound

import (
	"os/exec"
	"runtime"
)

// SoundType represents different sound events
type SoundType int

const (
	SoundSuccess SoundType = iota
	SoundError
	SoundWarning
	SoundNotification
	SoundComplete
)

// Player handles sound playback
type Player struct {
	enabled bool
}

// New creates a new sound player
func New(enabled bool) *Player {
	return &Player{enabled: enabled}
}

// SetEnabled enables or disables sound
func (p *Player) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// IsEnabled returns whether sound is enabled
func (p *Player) IsEnabled() bool {
	return p.enabled
}

// Play plays a sound of the given type
func (p *Player) Play(soundType SoundType) error {
	if !p.enabled {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return p.playMacOS(soundType)
	case "linux":
		return p.playLinux(soundType)
	default:
		// Sound not supported on this platform
		return nil
	}
}

// PlaySuccess plays the success sound
func (p *Player) PlaySuccess() error {
	return p.Play(SoundSuccess)
}

// PlayError plays the error sound
func (p *Player) PlayError() error {
	return p.Play(SoundError)
}

// PlayWarning plays the warning sound
func (p *Player) PlayWarning() error {
	return p.Play(SoundWarning)
}

// PlayNotification plays the notification sound
func (p *Player) PlayNotification() error {
	return p.Play(SoundNotification)
}

// PlayComplete plays the completion sound
func (p *Player) PlayComplete() error {
	return p.Play(SoundComplete)
}

// playMacOS plays system sounds on macOS using afplay
func (p *Player) playMacOS(soundType SoundType) error {
	soundPath := getMacOSSoundPath(soundType)
	if soundPath == "" {
		return nil
	}

	// Run in background so it doesn't block
	cmd := exec.Command("afplay", soundPath)
	return cmd.Start() // Don't wait for completion
}

// getMacOSSoundPath returns the system sound path for macOS
func getMacOSSoundPath(soundType SoundType) string {
	const soundDir = "/System/Library/Sounds/"

	switch soundType {
	case SoundSuccess:
		return soundDir + "Glass.aiff"
	case SoundError:
		return soundDir + "Basso.aiff"
	case SoundWarning:
		return soundDir + "Sosumi.aiff"
	case SoundNotification:
		return soundDir + "Pop.aiff"
	case SoundComplete:
		return soundDir + "Hero.aiff"
	default:
		return soundDir + "Pop.aiff"
	}
}

// playLinux plays sounds on Linux using paplay
func (p *Player) playLinux(soundType SoundType) error {
	soundPath := getLinuxSoundPath(soundType)
	if soundPath == "" {
		return nil
	}

	cmd := exec.Command("paplay", soundPath)
	return cmd.Start() // Don't wait for completion
}

// getLinuxSoundPath returns a system sound path for Linux
func getLinuxSoundPath(soundType SoundType) string {
	const soundDir = "/usr/share/sounds/freedesktop/stereo/"

	switch soundType {
	case SoundSuccess:
		return soundDir + "complete.oga"
	case SoundError:
		return soundDir + "dialog-error.oga"
	case SoundWarning:
		return soundDir + "dialog-warning.oga"
	case SoundNotification:
		return soundDir + "message.oga"
	case SoundComplete:
		return soundDir + "complete.oga"
	default:
		return soundDir + "message.oga"
	}
}
