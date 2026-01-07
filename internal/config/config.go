package config

import (
	"os"
	"path/filepath"
)

// Default configuration values
const (
	DefaultSprintStatus = "_bmad-output/implementation-artifacts/sprint-status.yaml"
	DefaultStoryDir     = "_bmad-output/implementation-artifacts"
	DefaultTimeout      = 600 // 10 minutes
	DefaultRetries      = 1
)

// Config holds all application configuration
type Config struct {
	// Paths
	SprintStatusPath string
	StoryDir         string
	WorkingDir       string

	// Execution settings
	Timeout int // seconds
	Retries int

	// UI settings
	Theme string

	// Feature flags
	SoundEnabled         bool
	NotificationsEnabled bool
}

// New creates a new Config with default values
func New() *Config {
	wd, _ := os.Getwd()

	return &Config{
		SprintStatusPath:     filepath.Join(wd, DefaultSprintStatus),
		StoryDir:             filepath.Join(wd, DefaultStoryDir),
		WorkingDir:           wd,
		Timeout:              DefaultTimeout,
		Retries:              DefaultRetries,
		Theme:                "catppuccin",
		SoundEnabled:         false,
		NotificationsEnabled: true,
	}
}

// StoryFilePath returns the full path for a story file
func (c *Config) StoryFilePath(storyKey string) string {
	return filepath.Join(c.StoryDir, storyKey+".md")
}

// StoryFileExists checks if a story file already exists
func (c *Config) StoryFileExists(storyKey string) bool {
	_, err := os.Stat(c.StoryFilePath(storyKey))
	return err == nil
}
