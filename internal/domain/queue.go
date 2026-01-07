package domain

import (
	"time"
)

// QueueStatus represents the status of the execution queue
type QueueStatus string

const (
	QueueIdle      QueueStatus = "idle"
	QueueRunning   QueueStatus = "running"
	QueuePaused    QueueStatus = "paused"
	QueueCompleted QueueStatus = "completed"
)

// QueueItem represents a story in the queue with its execution state
type QueueItem struct {
	Story     Story
	Status    ExecutionStatus
	Execution *Execution // Populated when executing/completed
	AddedAt   time.Time
	Position  int // Position in queue (1-based for display)
}

// Queue manages a list of stories to be executed
type Queue struct {
	Items   []*QueueItem
	Status  QueueStatus
	Current int // Index of currently executing item (-1 if none)

	// Timing and statistics
	StartTime time.Time
	EndTime   time.Time

	// Historical averages for ETA calculation (per step)
	StepAverages map[StepName]time.Duration
}

// NewQueue creates a new empty queue
func NewQueue() *Queue {
	return &Queue{
		Items:        make([]*QueueItem, 0),
		Status:       QueueIdle,
		Current:      -1,
		StepAverages: make(map[StepName]time.Duration),
	}
}

// Add adds a story to the queue
func (q *Queue) Add(story Story) {
	// Don't add duplicates
	for _, item := range q.Items {
		if item.Story.Key == story.Key {
			return
		}
	}

	q.Items = append(q.Items, &QueueItem{
		Story:    story,
		Status:   ExecutionPending,
		AddedAt:  time.Now(),
		Position: len(q.Items) + 1,
	})
	q.updatePositions()
}

// AddMultiple adds multiple stories to the queue
func (q *Queue) AddMultiple(stories []Story) {
	for _, story := range stories {
		q.Add(story)
	}
}

// Remove removes a story from the queue by key
func (q *Queue) Remove(key string) bool {
	for i, item := range q.Items {
		if item.Story.Key == key {
			// Can't remove if currently executing
			if i == q.Current && q.Status == QueueRunning {
				return false
			}
			q.Items = append(q.Items[:i], q.Items[i+1:]...)
			q.updatePositions()

			// Adjust current index if needed
			if q.Current > i {
				q.Current--
			}
			return true
		}
	}
	return false
}

// Clear removes all pending items from the queue (keeps completed/running)
func (q *Queue) Clear() {
	if q.Status == QueueRunning {
		// Only clear items after current
		if q.Current >= 0 && q.Current < len(q.Items)-1 {
			q.Items = q.Items[:q.Current+1]
		}
	} else {
		q.Items = make([]*QueueItem, 0)
		q.Current = -1
	}
	q.updatePositions()
}

// MoveUp moves an item up in the queue (only pending items)
func (q *Queue) MoveUp(index int) bool {
	if index <= 0 || index >= len(q.Items) {
		return false
	}

	// Can't reorder items that are executing or completed
	if q.Items[index].Status != ExecutionPending {
		return false
	}
	if q.Items[index-1].Status != ExecutionPending {
		return false
	}

	q.Items[index], q.Items[index-1] = q.Items[index-1], q.Items[index]
	q.updatePositions()
	return true
}

// MoveDown moves an item down in the queue (only pending items)
func (q *Queue) MoveDown(index int) bool {
	if index < 0 || index >= len(q.Items)-1 {
		return false
	}

	// Can't reorder items that are executing or completed
	if q.Items[index].Status != ExecutionPending {
		return false
	}
	if q.Items[index+1].Status != ExecutionPending {
		return false
	}

	q.Items[index], q.Items[index+1] = q.Items[index+1], q.Items[index]
	q.updatePositions()
	return true
}

// GetPending returns all pending items
func (q *Queue) GetPending() []*QueueItem {
	var pending []*QueueItem
	for _, item := range q.Items {
		if item.Status == ExecutionPending {
			pending = append(pending, item)
		}
	}
	return pending
}

// GetCompleted returns all completed items (success or failed)
func (q *Queue) GetCompleted() []*QueueItem {
	var completed []*QueueItem
	for _, item := range q.Items {
		if item.Status == ExecutionCompleted || item.Status == ExecutionFailed {
			completed = append(completed, item)
		}
	}
	return completed
}

// CurrentItem returns the currently executing item
func (q *Queue) CurrentItem() *QueueItem {
	if q.Current >= 0 && q.Current < len(q.Items) {
		return q.Items[q.Current]
	}
	return nil
}

// NextPending returns the next pending item for execution
func (q *Queue) NextPending() *QueueItem {
	for i, item := range q.Items {
		if item.Status == ExecutionPending {
			return q.Items[i]
		}
	}
	return nil
}

// TotalCount returns the total number of items
func (q *Queue) TotalCount() int {
	return len(q.Items)
}

// PendingCount returns the number of pending items
func (q *Queue) PendingCount() int {
	count := 0
	for _, item := range q.Items {
		if item.Status == ExecutionPending {
			count++
		}
	}
	return count
}

// CompletedCount returns the number of completed items
func (q *Queue) CompletedCount() int {
	count := 0
	for _, item := range q.Items {
		if item.Status == ExecutionCompleted {
			count++
		}
	}
	return count
}

// FailedCount returns the number of failed items
func (q *Queue) FailedCount() int {
	count := 0
	for _, item := range q.Items {
		if item.Status == ExecutionFailed {
			count++
		}
	}
	return count
}

// ProgressPercent returns overall queue progress as percentage
func (q *Queue) ProgressPercent() float64 {
	if len(q.Items) == 0 {
		return 0
	}

	completed := q.CompletedCount() + q.FailedCount()

	// Add partial progress from current item
	currentProgress := 0.0
	if current := q.CurrentItem(); current != nil && current.Execution != nil {
		currentProgress = current.Execution.ProgressPercent() / 100.0
	}

	return (float64(completed) + currentProgress) / float64(len(q.Items)) * 100
}

// EstimatedTimeRemaining calculates ETA based on historical averages
func (q *Queue) EstimatedTimeRemaining() time.Duration {
	if len(q.StepAverages) == 0 {
		// No history, use default estimate (5 min per step, 4 steps)
		pendingCount := q.PendingCount()
		return time.Duration(pendingCount) * 20 * time.Minute
	}

	// Calculate average total time per story
	var totalPerStory time.Duration
	for _, stepName := range AllSteps() {
		if avg, ok := q.StepAverages[stepName]; ok {
			totalPerStory += avg
		}
	}

	// Estimate for pending items
	pendingCount := q.PendingCount()
	remaining := time.Duration(pendingCount) * totalPerStory

	// Subtract elapsed time for current item
	if current := q.CurrentItem(); current != nil && current.Execution != nil {
		elapsed := time.Since(current.Execution.StartTime)
		if elapsed < totalPerStory {
			remaining -= elapsed
		}
	}

	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// UpdateStepAverage updates the average duration for a step
func (q *Queue) UpdateStepAverage(step StepName, duration time.Duration) {
	if existing, ok := q.StepAverages[step]; ok {
		// Simple moving average
		q.StepAverages[step] = (existing + duration) / 2
	} else {
		q.StepAverages[step] = duration
	}
}

// IsEmpty returns true if queue has no items
func (q *Queue) IsEmpty() bool {
	return len(q.Items) == 0
}

// HasPending returns true if there are pending items
func (q *Queue) HasPending() bool {
	return q.PendingCount() > 0
}

// Contains checks if a story is already in the queue
func (q *Queue) Contains(key string) bool {
	for _, item := range q.Items {
		if item.Story.Key == key {
			return true
		}
	}
	return false
}

// updatePositions updates the position field for all items
func (q *Queue) updatePositions() {
	for i, item := range q.Items {
		item.Position = i + 1
	}
}

// GetItem returns the item at the given index
func (q *Queue) GetItem(index int) *QueueItem {
	if index >= 0 && index < len(q.Items) {
		return q.Items[index]
	}
	return nil
}

// IndexOf returns the index of an item by key, or -1 if not found
func (q *Queue) IndexOf(key string) int {
	for i, item := range q.Items {
		if item.Story.Key == key {
			return i
		}
	}
	return -1
}
