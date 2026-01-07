package parser

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// SprintStatus represents the structure of sprint-status.yaml
type SprintStatus struct {
	DevelopmentStatus map[string]string `yaml:"development_status"`
}

// storyKeyPattern matches story keys like "3-1-user-auth"
var storyKeyPattern = regexp.MustCompile(`^\d+-\d+-.+$`)

// ParseSprintStatus parses the sprint-status.yaml file and returns stories
func ParseSprintStatus(cfg *config.Config) ([]domain.Story, error) {
	data, err := os.ReadFile(cfg.SprintStatusPath)
	if err != nil {
		return nil, err
	}

	var status SprintStatus
	if err := yaml.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	var stories []domain.Story

	for key, statusStr := range status.DevelopmentStatus {
		if !storyKeyPattern.MatchString(key) {
			continue
		}

		story := domain.Story{
			Key:        key,
			Epic:       extractEpic(key),
			Status:     domain.StoryStatus(statusStr),
			FilePath:   cfg.StoryFilePath(key),
			FileExists: cfg.StoryFileExists(key),
		}

		stories = append(stories, story)
	}

	// Sort stories by epic and then by key
	sort.Slice(stories, func(i, j int) bool {
		if stories[i].Epic != stories[j].Epic {
			return stories[i].Epic < stories[j].Epic
		}
		return stories[i].Key < stories[j].Key
	})

	return stories, nil
}

// extractEpic extracts the epic number from a story key (e.g., "3-1-story" -> 3)
func extractEpic(key string) int {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) > 0 {
		if epic, err := strconv.Atoi(parts[0]); err == nil {
			return epic
		}
	}
	return 0
}

// FilterStoriesByStatus returns stories with the given status
func FilterStoriesByStatus(stories []domain.Story, status domain.StoryStatus) []domain.Story {
	var filtered []domain.Story
	for _, s := range stories {
		if s.Status == status {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// FilterStoriesByEpic returns stories for the given epic
func FilterStoriesByEpic(stories []domain.Story, epic int) []domain.Story {
	if epic == 0 {
		return stories
	}
	var filtered []domain.Story
	for _, s := range stories {
		if s.Epic == epic {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// GetActionableStories returns stories that can be processed, in priority order
func GetActionableStories(stories []domain.Story) []domain.Story {
	var actionable []domain.Story

	// Priority order: in-progress > ready-for-dev > backlog
	priorities := []domain.StoryStatus{
		domain.StatusInProgress,
		domain.StatusReadyForDev,
		domain.StatusBacklog,
	}

	for _, status := range priorities {
		for _, s := range stories {
			if s.Status == status {
				actionable = append(actionable, s)
			}
		}
	}

	return actionable
}

// CountByStatus returns counts of stories by status
func CountByStatus(stories []domain.Story) map[domain.StoryStatus]int {
	counts := make(map[domain.StoryStatus]int)
	for _, s := range stories {
		counts[s.Status]++
	}
	return counts
}

// GetUniqueEpics returns a sorted list of unique epic numbers
func GetUniqueEpics(stories []domain.Story) []int {
	epicMap := make(map[int]bool)
	for _, s := range stories {
		epicMap[s.Epic] = true
	}

	var epics []int
	for e := range epicMap {
		epics = append(epics, e)
	}
	sort.Ints(epics)
	return epics
}
