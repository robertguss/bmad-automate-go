package messages

import (
	"github.com/robertguss/bmad-automate-go/internal/domain"
)

// Navigation messages
type NavigateMsg struct {
	View domain.View
}

type NavigateBackMsg struct{}

// Story messages
type StoriesLoadedMsg struct {
	Stories []domain.Story
	Error   error
}

type StorySelectedMsg struct {
	Story domain.Story
}

type StoriesFilteredMsg struct {
	Epic   int
	Status domain.StoryStatus
}

// Window size message
type WindowSizeMsg struct {
	Width  int
	Height int
}

// Error message
type ErrorMsg struct {
	Error error
}

// Tick message for animations/updates
type TickMsg struct{}
