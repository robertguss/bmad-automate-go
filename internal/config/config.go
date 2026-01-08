package config

import (
	"os"
	"path/filepath"
)

// Default configuration values
const (
	DefaultSprintStatus  = "_bmad-output/implementation-artifacts/sprint-status.yaml"
	DefaultStoryDir      = "_bmad-output/implementation-artifacts"
	DefaultTimeout       = 600 // 10 minutes
	DefaultRetries       = 1
	DefaultDataDir       = ".bmad"
	DefaultDBName        = "bmad.db"
	DefaultAPIPort       = 8080
	DefaultMaxWorkers    = 1
	DefaultWatchDebounce = 500 // milliseconds
)

// Config holds all application configuration
type Config struct {
	// Paths
	SprintStatusPath string
	StoryDir         string
	WorkingDir       string
	DataDir          string // Directory for app data (database, etc.)
	DatabasePath     string // Path to SQLite database

	// Execution settings
	Timeout int // seconds
	Retries int

	// UI settings
	Theme           string
	CustomThemePath string // Path to custom theme YAML file

	// Feature flags
	SoundEnabled         bool
	NotificationsEnabled bool

	// Phase 6: Profile settings
	ActiveProfile string // Name of active profile

	// Phase 6: Workflow settings
	ActiveWorkflow string // Name of active workflow (default: "default")

	// Phase 6: Watch mode settings
	WatchEnabled  bool // Enable file watching
	WatchDebounce int  // Debounce time in milliseconds

	// Phase 6: Parallel execution settings
	MaxWorkers      int  // Max parallel workers (1 = sequential)
	ParallelEnabled bool // Enable parallel execution

	// Phase 6: API server settings
	APIEnabled bool // Enable REST API server
	APIPort    int  // Port for API server

	// Security settings
	APIKey             string   // API key for authentication (optional, from BMAD_API_KEY env)
	CORSAllowedOrigins []string // Allowed CORS origins (empty = localhost only)
}

// New creates a new Config with default values
func New() *Config {
	wd, _ := os.Getwd()
	dataDir := filepath.Join(wd, DefaultDataDir)

	return &Config{
		SprintStatusPath:     filepath.Join(wd, DefaultSprintStatus),
		StoryDir:             filepath.Join(wd, DefaultStoryDir),
		WorkingDir:           wd,
		DataDir:              dataDir,
		DatabasePath:         filepath.Join(dataDir, DefaultDBName),
		Timeout:              DefaultTimeout,
		Retries:              DefaultRetries,
		Theme:                "catppuccin",
		SoundEnabled:         false,
		NotificationsEnabled: true,
		ActiveProfile:        "",
		ActiveWorkflow:       "default",
		WatchEnabled:         false,
		WatchDebounce:        DefaultWatchDebounce,
		MaxWorkers:           DefaultMaxWorkers,
		ParallelEnabled:      false,
		APIEnabled:           false,
		APIPort:              DefaultAPIPort,
		APIKey:               os.Getenv("BMAD_API_KEY"),
		CORSAllowedOrigins:   defaultCORSOrigins(),
	}
}

// defaultCORSOrigins returns the default CORS origins based on environment
func defaultCORSOrigins() []string {
	if origins := os.Getenv("BMAD_CORS_ORIGINS"); origins != "" {
		// Split comma-separated origins from environment
		return splitOrigins(origins)
	}
	// Default to localhost only (secure default)
	return []string{
		"http://localhost:*",
		"http://127.0.0.1:*",
	}
}

// splitOrigins splits a comma-separated string of origins
func splitOrigins(origins string) []string {
	var result []string
	for _, origin := range filepath.SplitList(origins) {
		origin = trimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	// If the string contains commas but not path separators, split by comma
	if len(result) <= 1 && len(origins) > 0 {
		result = nil
		for _, part := range splitByComma(origins) {
			part = trimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

// splitByComma splits a string by commas
func splitByComma(s string) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// EnsureDataDir creates the data directory if it doesn't exist
func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
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
