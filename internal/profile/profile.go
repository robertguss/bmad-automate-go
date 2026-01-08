package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile represents a project configuration profile
type Profile struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description,omitempty"`
	SprintStatusPath string `yaml:"sprint_status_path,omitempty"`
	StoryDir         string `yaml:"story_dir,omitempty"`
	WorkingDir       string `yaml:"working_dir,omitempty"`
	Timeout          int    `yaml:"timeout,omitempty"`
	Retries          int    `yaml:"retries,omitempty"`
	Theme            string `yaml:"theme,omitempty"`
	Workflow         string `yaml:"workflow,omitempty"` // Name of custom workflow to use
	MaxWorkers       int    `yaml:"max_workers,omitempty"`
}

// ProfileStore manages profile persistence
type ProfileStore struct {
	profileDir string
	profiles   map[string]*Profile
	active     string
}

// NewProfileStore creates a new profile store
func NewProfileStore(dataDir string) *ProfileStore {
	profileDir := filepath.Join(dataDir, "profiles")
	return &ProfileStore{
		profileDir: profileDir,
		profiles:   make(map[string]*Profile),
	}
}

// Load loads all profiles from disk
func (ps *ProfileStore) Load() error {
	if err := os.MkdirAll(ps.profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Load profiles from YAML files
	files, err := filepath.Glob(filepath.Join(ps.profileDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	for _, file := range files {
		profile, err := ps.loadProfile(file)
		if err != nil {
			continue // Skip invalid profiles
		}
		ps.profiles[profile.Name] = profile
	}

	// Load active profile marker
	activeFile := filepath.Join(ps.profileDir, ".active")
	if data, err := os.ReadFile(activeFile); err == nil {
		ps.active = string(data)
	}

	return nil
}

// loadProfile loads a single profile from a YAML file
func (ps *ProfileStore) loadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	// Use filename as name if not specified
	if profile.Name == "" {
		base := filepath.Base(path)
		profile.Name = base[:len(base)-5] // Remove .yaml extension
	}

	return &profile, nil
}

// validateProfileName checks for path traversal attempts in profile names
// SEC-008: Prevents directory traversal attacks via malicious profile names
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	// Reject path separators and traversal sequences
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("profile name contains invalid characters: must not contain /, \\, or ..")
	}
	// Also reject names that start with a dot (hidden files)
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("profile name cannot start with a dot")
	}
	return nil
}

// Save saves a profile to disk
func (ps *ProfileStore) Save(profile *Profile) error {
	// SEC-008: Validate profile name to prevent path traversal
	if err := validateProfileName(profile.Name); err != nil {
		return err
	}

	if err := os.MkdirAll(ps.profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	path := filepath.Join(ps.profileDir, profile.Name+".yaml")
	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	ps.profiles[profile.Name] = profile
	return nil
}

// Delete removes a profile from disk
func (ps *ProfileStore) Delete(name string) error {
	// SEC-008: Validate profile name to prevent path traversal
	if err := validateProfileName(name); err != nil {
		return err
	}

	path := filepath.Join(ps.profileDir, name+".yaml")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	delete(ps.profiles, name)
	return nil
}

// Get returns a profile by name
func (ps *ProfileStore) Get(name string) (*Profile, bool) {
	p, ok := ps.profiles[name]
	return p, ok
}

// List returns all profile names
func (ps *ProfileStore) List() []string {
	names := make([]string, 0, len(ps.profiles))
	for name := range ps.profiles {
		names = append(names, name)
	}
	return names
}

// GetAll returns all profiles
func (ps *ProfileStore) GetAll() []*Profile {
	profiles := make([]*Profile, 0, len(ps.profiles))
	for _, p := range ps.profiles {
		profiles = append(profiles, p)
	}
	return profiles
}

// SetActive sets the active profile
func (ps *ProfileStore) SetActive(name string) error {
	if _, ok := ps.profiles[name]; !ok {
		return fmt.Errorf("profile not found: %s", name)
	}

	activeFile := filepath.Join(ps.profileDir, ".active")
	if err := os.WriteFile(activeFile, []byte(name), 0644); err != nil {
		return fmt.Errorf("failed to set active profile: %w", err)
	}

	ps.active = name
	return nil
}

// GetActive returns the active profile name
func (ps *ProfileStore) GetActive() string {
	return ps.active
}

// GetActiveProfile returns the active profile
func (ps *ProfileStore) GetActiveProfile() *Profile {
	if ps.active == "" {
		return nil
	}
	return ps.profiles[ps.active]
}

// CreateDefault creates a default profile from current config
func (ps *ProfileStore) CreateDefault(sprintStatus, storyDir, workingDir string, timeout, retries int, theme string) *Profile {
	return &Profile{
		Name:             "default",
		Description:      "Default project configuration",
		SprintStatusPath: sprintStatus,
		StoryDir:         storyDir,
		WorkingDir:       workingDir,
		Timeout:          timeout,
		Retries:          retries,
		Theme:            theme,
	}
}
