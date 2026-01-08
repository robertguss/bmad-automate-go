package watcher

import (
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// RefreshMsg is sent when watched files change
type RefreshMsg struct {
	Path string
}

// ErrorMsg is sent when watcher encounters an error
type ErrorMsg struct {
	Error error
}

// Watcher monitors files for changes and sends refresh messages
type Watcher struct {
	watcher  *fsnotify.Watcher
	program  *tea.Program
	paths    []string
	debounce time.Duration

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}

	// Debounce tracking
	lastEvent time.Time
	pending   bool
}

// New creates a new file watcher
func New(debounce time.Duration) *Watcher {
	return &Watcher{
		debounce: debounce,
		paths:    make([]string, 0),
		stopCh:   make(chan struct{}),
	}
}

// SetProgram sets the tea.Program for sending messages
func (w *Watcher) SetProgram(p *tea.Program) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.program = p
}

// AddPath adds a path to watch
func (w *Watcher) AddPath(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paths = append(w.paths, path)

	if w.watcher != nil && w.running {
		_ = w.watcher.Add(path)
	}
}

// AddPaths adds multiple paths to watch
func (w *Watcher) AddPaths(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paths = append(w.paths, paths...)

	if w.watcher != nil && w.running {
		for _, path := range paths {
			_ = w.watcher.Add(path)
		}
	}
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}

	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		w.mu.Unlock()
		return err
	}

	// Add all configured paths
	for _, path := range w.paths {
		// Watch the directory containing the file for better reliability
		dir := filepath.Dir(path)
		_ = w.watcher.Add(dir)
	}

	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	go w.run()
	return nil
}

// Stop stops watching for file changes
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.running = false
	close(w.stopCh)

	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// IsRunning returns whether the watcher is currently active
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// run is the main event loop
func (w *Watcher) run() {
	debounceTimer := time.NewTimer(w.debounce)
	debounceTimer.Stop()

	for {
		select {
		case <-w.stopCh:
			debounceTimer.Stop()
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if this is a file we're interested in
			if !w.isWatchedPath(event.Name) {
				continue
			}

			// Only react to write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Reset debounce timer
			w.mu.Lock()
			w.pending = true
			w.lastEvent = time.Now()
			w.mu.Unlock()

			debounceTimer.Reset(w.debounce)

		case <-debounceTimer.C:
			w.mu.Lock()
			pending := w.pending
			w.pending = false
			w.mu.Unlock()

			if pending {
				w.sendMsg(RefreshMsg{})
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.sendMsg(ErrorMsg{Error: err})
		}
	}
}

// isWatchedPath checks if the given path matches any watched path
func (w *Watcher) isWatchedPath(path string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get absolute path for comparison
	absPath, _ := filepath.Abs(path)

	for _, watchedPath := range w.paths {
		absWatched, _ := filepath.Abs(watchedPath)
		if absPath == absWatched {
			return true
		}
		// Also check by base name for reliability
		if filepath.Base(path) == filepath.Base(watchedPath) {
			return true
		}
	}
	return false
}

// sendMsg safely sends a message to the tea.Program
func (w *Watcher) sendMsg(msg tea.Msg) {
	w.mu.Lock()
	program := w.program
	w.mu.Unlock()

	if program != nil {
		program.Send(msg)
	}
}

// WatchSprintStatus creates a watcher configured for sprint-status.yaml
func WatchSprintStatus(sprintStatusPath string, debounce time.Duration) *Watcher {
	w := New(debounce)
	w.AddPath(sprintStatusPath)
	return w
}
