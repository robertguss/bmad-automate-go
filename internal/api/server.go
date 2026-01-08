package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robertguss/bmad-automate-go/internal/config"
	"github.com/robertguss/bmad-automate-go/internal/domain"
	"github.com/robertguss/bmad-automate-go/internal/executor"
	"github.com/robertguss/bmad-automate-go/internal/parser"
	"github.com/robertguss/bmad-automate-go/internal/storage"
)

// Server is the REST API server
type Server struct {
	config        *config.Config
	storage       storage.Storage
	executor      *executor.Executor
	batchExecutor *executor.BatchExecutor
	wsHub         *WebSocketHub

	mu      sync.RWMutex
	stories []domain.Story
	server  *http.Server
	running bool
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, store storage.Storage, exec *executor.Executor, batchExec *executor.BatchExecutor) *Server {
	wsHub := NewWebSocketHub()
	// Configure WebSocket security settings (SEC-005/006)
	wsHub.SetSecurityConfig(cfg.APIKey, cfg.CORSAllowedOrigins)

	return &Server{
		config:        cfg,
		storage:       store,
		executor:      exec,
		batchExecutor: batchExec,
		wsHub:         wsHub,
	}
}

// SetStories sets the current stories list
func (s *Server) SetStories(stories []domain.Story) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stories = stories
}

// GetWebSocketHub returns the WebSocket hub
func (s *Server) GetWebSocketHub() *WebSocketHub {
	return s.wsHub
}

// Start starts the API server on the given port
func (s *Server) Start(port int) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	router := s.setupRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start WebSocket hub
	go s.wsHub.Run()

	return s.server.ListenAndServe()
}

// Stop stops the API server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	s.wsHub.Stop()

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware(s.config.CORSAllowedOrigins))

	// Health check (public, no auth required)
	r.Get("/health", s.healthHandler)

	// API routes (protected by API key if configured)
	r.Route("/api", func(r chi.Router) {
		// Apply API key authentication to all /api routes
		r.Use(apiKeyAuthMiddleware(s.config.APIKey))
		// Stories
		r.Get("/stories", s.listStoriesHandler)
		r.Get("/stories/{key}", s.getStoryHandler)
		r.Post("/stories/refresh", s.refreshStoriesHandler)

		// Queue management
		r.Get("/queue", s.getQueueHandler)
		r.Post("/queue/add", s.addToQueueHandler)
		r.Post("/queue/add/{key}", s.addStoryToQueueHandler)
		r.Delete("/queue/{key}", s.removeFromQueueHandler)
		r.Post("/queue/clear", s.clearQueueHandler)
		r.Post("/queue/reorder", s.reorderQueueHandler)

		// Execution control
		r.Get("/execution", s.getExecutionHandler)
		r.Post("/execution/start", s.startExecutionHandler)
		r.Post("/execution/start/{key}", s.startStoryExecutionHandler)
		r.Post("/execution/pause", s.pauseExecutionHandler)
		r.Post("/execution/resume", s.resumeExecutionHandler)
		r.Post("/execution/cancel", s.cancelExecutionHandler)
		r.Post("/execution/skip", s.skipStepHandler)

		// History
		r.Get("/history", s.listHistoryHandler)
		r.Get("/history/{id}", s.getHistoryHandler)

		// Statistics
		r.Get("/stats", s.getStatsHandler)

		// Configuration
		r.Get("/config", s.getConfigHandler)

		// WebSocket endpoint
		r.Get("/ws", s.websocketHandler)
	})

	return r
}

// corsMiddleware creates CORS middleware with the given allowed origins
// SEC-003 fix: No longer uses "*" - requires explicit origin configuration
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	// Build a map for O(1) lookup, and track patterns with wildcards
	exactOrigins := make(map[string]bool)
	var patterns []string

	for _, origin := range allowedOrigins {
		if strings.Contains(origin, "*") {
			patterns = append(patterns, origin)
		} else {
			exactOrigins[origin] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if origin != "" {
				// Check exact match first
				if exactOrigins[origin] {
					allowed = true
				} else {
					// Check patterns (e.g., "http://localhost:*")
					for _, pattern := range patterns {
						if matchOriginPattern(origin, pattern) {
							allowed = true
							break
						}
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-API-Key")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyAuthMiddleware creates middleware that validates API key from header
// SEC-004 fix: Adds authentication to protect API endpoints
func apiKeyAuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no API key is configured, allow all requests (optional auth)
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for API key in header
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				// Also check Authorization header as Bearer token
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					providedKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if providedKey != apiKey {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// matchOriginPattern checks if an origin matches a pattern with wildcards
// e.g., "http://localhost:3000" matches "http://localhost:*"
func matchOriginPattern(origin, pattern string) bool {
	// Handle simple wildcard at the end (e.g., "http://localhost:*")
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(origin, prefix)
	}
	// Handle wildcard subdomain (e.g., "*.example.com")
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		// Extract host from origin (e.g., "https://sub.example.com" -> "sub.example.com")
		parts := strings.SplitN(origin, "://", 2)
		if len(parts) == 2 {
			host := strings.Split(parts[1], "/")[0]
			host = strings.Split(host, ":")[0] // Remove port
			return strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(suffix, ".")
		}
	}
	return false
}

// Response helpers

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Handlers

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) listStoriesHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	stories := s.stories
	s.mu.RUnlock()

	// Optional filtering
	epic := r.URL.Query().Get("epic")
	status := r.URL.Query().Get("status")

	filtered := make([]domain.Story, 0)
	for _, story := range stories {
		if epic != "" {
			if e, err := strconv.Atoi(epic); err == nil && story.Epic != e {
				continue
			}
		}
		if status != "" && string(story.Status) != status {
			continue
		}
		filtered = append(filtered, story)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stories": filtered,
		"count":   len(filtered),
	})
}

func (s *Server) getStoryHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	s.mu.RLock()
	var found *domain.Story
	for _, story := range s.stories {
		if story.Key == key {
			found = &story
			break
		}
	}
	s.mu.RUnlock()

	if found == nil {
		respondError(w, http.StatusNotFound, "story not found")
		return
	}

	respondJSON(w, http.StatusOK, found)
}

func (s *Server) refreshStoriesHandler(w http.ResponseWriter, r *http.Request) {
	stories, err := parser.ParseSprintStatus(s.config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.Lock()
	s.stories = stories
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stories": stories,
		"count":   len(stories),
	})
}

func (s *Server) getQueueHandler(w http.ResponseWriter, r *http.Request) {
	queue := s.batchExecutor.GetQueue()

	items := make([]map[string]interface{}, 0)
	for _, item := range queue.Items {
		items = append(items, map[string]interface{}{
			"story":    item.Story,
			"status":   item.Status,
			"position": item.Position,
			"added_at": item.AddedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":   items,
		"status":  queue.Status,
		"current": queue.Current,
		"total":   len(queue.Items),
		"pending": queue.PendingCount(),
		"eta":     queue.EstimatedTimeRemaining().Seconds(),
	})
}

func (s *Server) addToQueueHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keys []string `json:"keys"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.mu.RLock()
	stories := make([]domain.Story, 0)
	for _, key := range req.Keys {
		for _, story := range s.stories {
			if story.Key == key {
				stories = append(stories, story)
				break
			}
		}
	}
	s.mu.RUnlock()

	if len(stories) == 0 {
		respondError(w, http.StatusBadRequest, "no valid stories found")
		return
	}

	s.batchExecutor.AddToQueue(stories)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"added": len(stories),
		"queue": s.batchExecutor.GetQueue().TotalCount(),
	})
}

func (s *Server) addStoryToQueueHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	s.mu.RLock()
	var found *domain.Story
	for _, story := range s.stories {
		if story.Key == key {
			found = &story
			break
		}
	}
	s.mu.RUnlock()

	if found == nil {
		respondError(w, http.StatusNotFound, "story not found")
		return
	}

	s.batchExecutor.AddToQueue([]domain.Story{*found})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"added": 1,
		"queue": s.batchExecutor.GetQueue().TotalCount(),
	})
}

func (s *Server) removeFromQueueHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	queue := s.batchExecutor.GetQueue()
	queue.Remove(key)

	respondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Server) clearQueueHandler(w http.ResponseWriter, r *http.Request) {
	queue := s.batchExecutor.GetQueue()
	queue.Clear()

	respondJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) reorderQueueHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Index     int    `json:"index"`
		Direction string `json:"direction"` // "up" or "down"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	queue := s.batchExecutor.GetQueue()
	switch req.Direction {
	case "up":
		queue.MoveUp(req.Index)
	case "down":
		queue.MoveDown(req.Index)
	default:
		respondError(w, http.StatusBadRequest, "invalid direction")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "reordered"})
}

func (s *Server) getExecutionHandler(w http.ResponseWriter, r *http.Request) {
	exec := s.executor.GetExecution()
	if exec == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"running": false,
		})
		return
	}

	steps := make([]map[string]interface{}, 0)
	for _, step := range exec.Steps {
		steps = append(steps, map[string]interface{}{
			"name":     step.Name,
			"status":   step.Status,
			"duration": step.Duration.Seconds(),
			"attempt":  step.Attempt,
			"error":    step.Error,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"running":  exec.Status == domain.ExecutionRunning,
		"status":   exec.Status,
		"story":    exec.Story,
		"current":  exec.Current,
		"steps":    steps,
		"duration": time.Since(exec.StartTime).Seconds(),
		"progress": exec.ProgressPercent(),
	})
}

func (s *Server) startExecutionHandler(w http.ResponseWriter, r *http.Request) {
	queue := s.batchExecutor.GetQueue()
	if !queue.HasPending() {
		respondError(w, http.StatusBadRequest, "no items in queue")
		return
	}

	if s.batchExecutor.IsRunning() {
		respondError(w, http.StatusConflict, "execution already running")
		return
	}

	// Start in background
	go s.batchExecutor.Start()

	respondJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) startStoryExecutionHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	s.mu.RLock()
	var found *domain.Story
	for _, story := range s.stories {
		if story.Key == key {
			found = &story
			break
		}
	}
	s.mu.RUnlock()

	if found == nil {
		respondError(w, http.StatusNotFound, "story not found")
		return
	}

	if s.executor.GetExecution() != nil &&
		s.executor.GetExecution().Status == domain.ExecutionRunning {
		respondError(w, http.StatusConflict, "execution already running")
		return
	}

	// Start execution in background
	go s.executor.Execute(*found)

	respondJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) pauseExecutionHandler(w http.ResponseWriter, r *http.Request) {
	if s.batchExecutor.IsRunning() {
		s.batchExecutor.Pause()
	} else if exec := s.executor.GetExecution(); exec != nil && exec.Status == domain.ExecutionRunning {
		s.executor.Pause()
	} else {
		respondError(w, http.StatusBadRequest, "no execution running")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) resumeExecutionHandler(w http.ResponseWriter, r *http.Request) {
	if s.batchExecutor.IsPaused() {
		s.batchExecutor.Resume()
	} else if exec := s.executor.GetExecution(); exec != nil && exec.Status == domain.ExecutionPaused {
		s.executor.Resume()
	} else {
		respondError(w, http.StatusBadRequest, "no execution paused")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) cancelExecutionHandler(w http.ResponseWriter, r *http.Request) {
	if s.batchExecutor.IsRunning() {
		s.batchExecutor.Cancel()
	} else if exec := s.executor.GetExecution(); exec != nil {
		s.executor.Cancel()
	} else {
		respondError(w, http.StatusBadRequest, "no execution to cancel")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) skipStepHandler(w http.ResponseWriter, r *http.Request) {
	if exec := s.executor.GetExecution(); exec != nil && exec.Status == domain.ExecutionRunning {
		s.executor.Skip()
		respondJSON(w, http.StatusOK, map[string]string{"status": "skipping"})
		return
	}

	respondError(w, http.StatusBadRequest, "no step to skip")
}

func (s *Server) listHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		respondError(w, http.StatusServiceUnavailable, "storage not available")
		return
	}

	// Parse query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	filter := &storage.ExecutionFilter{
		Limit: limit,
	}

	if q := r.URL.Query().Get("story"); q != "" {
		filter.StoryKey = q
	}

	if e := r.URL.Query().Get("epic"); e != "" {
		if epic, err := strconv.Atoi(e); err == nil {
			filter.Epic = &epic
		}
	}

	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = domain.ExecutionStatus(s)
	}

	records, err := s.storage.ListExecutions(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	executions := make([]map[string]interface{}, 0)
	for _, rec := range records {
		executions = append(executions, map[string]interface{}{
			"id":         rec.ID,
			"story_key":  rec.StoryKey,
			"story_epic": rec.StoryEpic,
			"status":     rec.Status,
			"start_time": rec.StartTime,
			"duration":   rec.Duration.Seconds(),
			"error":      rec.Error,
		})
	}

	count, _ := s.storage.CountExecutions(r.Context(), filter)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
		"total":      count,
	})
}

func (s *Server) getHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		respondError(w, http.StatusServiceUnavailable, "storage not available")
		return
	}

	id := chi.URLParam(r, "id")
	record, err := s.storage.GetExecutionWithOutput(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "execution not found")
		return
	}

	steps := make([]map[string]interface{}, 0)
	for _, step := range record.Steps {
		steps = append(steps, map[string]interface{}{
			"name":     step.StepName,
			"status":   step.Status,
			"duration": step.Duration.Seconds(),
			"attempt":  step.Attempt,
			"command":  step.Command,
			"error":    step.Error,
			"output":   step.Output,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         record.ID,
		"story_key":  record.StoryKey,
		"story_epic": record.StoryEpic,
		"status":     record.Status,
		"start_time": record.StartTime,
		"end_time":   record.EndTime,
		"duration":   record.Duration.Seconds(),
		"error":      record.Error,
		"steps":      steps,
	})
}

func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		respondError(w, http.StatusServiceUnavailable, "storage not available")
		return
	}

	stats, err := s.storage.GetStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stepStats := make(map[string]interface{})
	for name, ss := range stats.StepStats {
		stepStats[string(name)] = map[string]interface{}{
			"total":        ss.TotalCount,
			"success":      ss.SuccessCount,
			"failure":      ss.FailureCount,
			"skipped":      ss.SkippedCount,
			"success_rate": ss.SuccessRate,
			"avg_duration": ss.AvgDuration.Seconds(),
			"min_duration": ss.MinDuration.Seconds(),
			"max_duration": ss.MaxDuration.Seconds(),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_executions":   stats.TotalExecutions,
		"successful":         stats.SuccessfulCount,
		"failed":             stats.FailedCount,
		"cancelled":          stats.CancelledCount,
		"success_rate":       stats.SuccessRate,
		"avg_duration":       stats.AvgDuration.Seconds(),
		"total_duration":     stats.TotalDuration.Seconds(),
		"step_stats":         stepStats,
		"executions_by_day":  stats.ExecutionsByDay,
		"executions_by_epic": stats.ExecutionsByEpic,
	})
}

func (s *Server) getConfigHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"working_dir":   s.config.WorkingDir,
		"sprint_status": s.config.SprintStatusPath,
		"story_dir":     s.config.StoryDir,
		"timeout":       s.config.Timeout,
		"retries":       s.config.Retries,
		"theme":         s.config.Theme,
		"sound_enabled": s.config.SoundEnabled,
		"notifications": s.config.NotificationsEnabled,
	})
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	s.wsHub.ServeWs(w, r)
}

// BroadcastMessage sends a message to all connected WebSocket clients
func (s *Server) BroadcastMessage(msgType string, data interface{}) {
	msg := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
	s.wsHub.Broadcast(msg)
}
