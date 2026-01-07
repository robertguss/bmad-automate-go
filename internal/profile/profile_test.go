package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfileStore(t *testing.T) {
	tempDir := t.TempDir()

	store := NewProfileStore(tempDir)

	require.NotNil(t, store)
	assert.Equal(t, filepath.Join(tempDir, "profiles"), store.profileDir)
	assert.NotNil(t, store.profiles)
	assert.Len(t, store.profiles, 0)
	assert.Empty(t, store.active)
}

func TestProfileStore_Load(t *testing.T) {
	t.Run("creates profile directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)

		err := store.Load()
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(tempDir, "profiles"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("loads profiles from files", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)

		profileYAML := `name: test-profile
description: Test profile
timeout: 300
retries: 2
theme: nord
`
		os.WriteFile(filepath.Join(profileDir, "test-profile.yaml"), []byte(profileYAML), 0644)

		store := NewProfileStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		p, ok := store.Get("test-profile")
		assert.True(t, ok)
		assert.NotNil(t, p)
		assert.Equal(t, "Test profile", p.Description)
		assert.Equal(t, 300, p.Timeout)
		assert.Equal(t, 2, p.Retries)
		assert.Equal(t, "nord", p.Theme)
	})

	t.Run("loads active profile marker", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)

		// Create a profile
		os.WriteFile(filepath.Join(profileDir, "active-test.yaml"), []byte("name: active-test"), 0644)
		// Create active marker
		os.WriteFile(filepath.Join(profileDir, ".active"), []byte("active-test"), 0644)

		store := NewProfileStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		assert.Equal(t, "active-test", store.GetActive())
	})

	t.Run("skips invalid profile files", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)

		// Write an invalid YAML file
		os.WriteFile(filepath.Join(profileDir, "invalid.yaml"), []byte("invalid: yaml: here"), 0644)

		store := NewProfileStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		// No profiles should be loaded
		assert.Len(t, store.List(), 0)
	})

	t.Run("uses filename as name if not specified", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)

		profileNoName := `description: Profile without name
timeout: 100
`
		os.WriteFile(filepath.Join(profileDir, "unnamed.yaml"), []byte(profileNoName), 0644)

		store := NewProfileStore(tempDir)
		err := store.Load()
		require.NoError(t, err)

		p, ok := store.Get("unnamed")
		assert.True(t, ok)
		assert.NotNil(t, p)
	})
}

func TestProfileStore_Save(t *testing.T) {
	t.Run("saves profile to disk", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		profile := &Profile{
			Name:        "new-profile",
			Description: "New profile",
			Timeout:     600,
		}

		err := store.Save(profile)
		require.NoError(t, err)

		// Verify file exists
		path := filepath.Join(tempDir, "profiles", "new-profile.yaml")
		_, err = os.Stat(path)
		assert.NoError(t, err)

		// Verify profile is in store
		p, ok := store.Get("new-profile")
		assert.True(t, ok)
		assert.NotNil(t, p)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		// Don't call Load - directory doesn't exist yet

		profile := &Profile{
			Name: "test-profile",
		}

		err := store.Save(profile)
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(filepath.Join(tempDir, "profiles"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("overwrites existing profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		// Save initial profile
		profile := &Profile{
			Name:    "overwrite-test",
			Timeout: 100,
		}
		store.Save(profile)

		// Update and save again
		profile.Timeout = 200
		err := store.Save(profile)
		require.NoError(t, err)

		// Verify updated value
		p, _ := store.Get("overwrite-test")
		assert.Equal(t, 200, p.Timeout)
	})
}

func TestProfileStore_Delete(t *testing.T) {
	t.Run("deletes existing profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		// First save a profile
		profile := &Profile{Name: "to-delete"}
		store.Save(profile)

		// Verify it exists
		_, ok := store.Get("to-delete")
		require.True(t, ok)

		// Delete it
		err := store.Delete("to-delete")
		require.NoError(t, err)

		// Verify it's gone from store
		_, ok = store.Get("to-delete")
		assert.False(t, ok)

		// Verify file is deleted
		path := filepath.Join(tempDir, "profiles", "to-delete.yaml")
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("succeeds for non-existent profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		err := store.Delete("nonexistent")
		assert.NoError(t, err)
	})
}

func TestProfileStore_Get(t *testing.T) {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, "profiles")
	os.MkdirAll(profileDir, 0755)
	os.WriteFile(filepath.Join(profileDir, "test.yaml"), []byte("name: test"), 0644)

	store := NewProfileStore(tempDir)
	store.Load()

	t.Run("returns profile when found", func(t *testing.T) {
		p, ok := store.Get("test")
		assert.True(t, ok)
		assert.NotNil(t, p)
		assert.Equal(t, "test", p.Name)
	})

	t.Run("returns false when not found", func(t *testing.T) {
		p, ok := store.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, p)
	})
}

func TestProfileStore_List(t *testing.T) {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, "profiles")
	os.MkdirAll(profileDir, 0755)
	os.WriteFile(filepath.Join(profileDir, "profile1.yaml"), []byte("name: profile1"), 0644)
	os.WriteFile(filepath.Join(profileDir, "profile2.yaml"), []byte("name: profile2"), 0644)

	store := NewProfileStore(tempDir)
	store.Load()

	names := store.List()

	assert.Len(t, names, 2)
	assert.Contains(t, names, "profile1")
	assert.Contains(t, names, "profile2")
}

func TestProfileStore_GetAll(t *testing.T) {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, "profiles")
	os.MkdirAll(profileDir, 0755)
	os.WriteFile(filepath.Join(profileDir, "profile1.yaml"), []byte("name: profile1"), 0644)
	os.WriteFile(filepath.Join(profileDir, "profile2.yaml"), []byte("name: profile2"), 0644)

	store := NewProfileStore(tempDir)
	store.Load()

	profiles := store.GetAll()

	assert.Len(t, profiles, 2)
}

func TestProfileStore_SetActive(t *testing.T) {
	t.Run("sets active profile", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)
		os.WriteFile(filepath.Join(profileDir, "test.yaml"), []byte("name: test"), 0644)

		store := NewProfileStore(tempDir)
		store.Load()

		err := store.SetActive("test")
		require.NoError(t, err)

		assert.Equal(t, "test", store.GetActive())

		// Verify file was written
		data, err := os.ReadFile(filepath.Join(profileDir, ".active"))
		require.NoError(t, err)
		assert.Equal(t, "test", string(data))
	})

	t.Run("returns error for non-existent profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		err := store.SetActive("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})
}

func TestProfileStore_GetActive(t *testing.T) {
	t.Run("returns empty string when no active profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		assert.Empty(t, store.GetActive())
	})

	t.Run("returns active profile name", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)
		os.WriteFile(filepath.Join(profileDir, "test.yaml"), []byte("name: test"), 0644)

		store := NewProfileStore(tempDir)
		store.Load()
		store.SetActive("test")

		assert.Equal(t, "test", store.GetActive())
	})
}

func TestProfileStore_GetActiveProfile(t *testing.T) {
	t.Run("returns nil when no active profile", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewProfileStore(tempDir)
		store.Load()

		assert.Nil(t, store.GetActiveProfile())
	})

	t.Run("returns active profile", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, "profiles")
		os.MkdirAll(profileDir, 0755)
		os.WriteFile(filepath.Join(profileDir, "test.yaml"), []byte("name: test\ndescription: Test profile"), 0644)

		store := NewProfileStore(tempDir)
		store.Load()
		store.SetActive("test")

		p := store.GetActiveProfile()
		require.NotNil(t, p)
		assert.Equal(t, "test", p.Name)
		assert.Equal(t, "Test profile", p.Description)
	})
}

func TestProfileStore_CreateDefault(t *testing.T) {
	tempDir := t.TempDir()
	store := NewProfileStore(tempDir)

	profile := store.CreateDefault(
		"/path/to/sprint-status.yaml",
		"/path/to/stories",
		"/path/to/workdir",
		600,
		3,
		"catppuccin",
	)

	assert.Equal(t, "default", profile.Name)
	assert.Equal(t, "Default project configuration", profile.Description)
	assert.Equal(t, "/path/to/sprint-status.yaml", profile.SprintStatusPath)
	assert.Equal(t, "/path/to/stories", profile.StoryDir)
	assert.Equal(t, "/path/to/workdir", profile.WorkingDir)
	assert.Equal(t, 600, profile.Timeout)
	assert.Equal(t, 3, profile.Retries)
	assert.Equal(t, "catppuccin", profile.Theme)
}
