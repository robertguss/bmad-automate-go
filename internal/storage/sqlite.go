package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/robertguss/bmad-automate-go/internal/domain"
)

// SQLiteStorage implements Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = MEMORY",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	s := &SQLiteStorage{db: db}

	// Run migrations
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

// NewInMemoryStorage creates an in-memory SQLite storage (for testing)
func NewInMemoryStorage() (*SQLiteStorage, error) {
	return NewSQLiteStorage(":memory:")
}

// migrate runs database migrations
func (s *SQLiteStorage) migrate() error {
	_, err := s.db.Exec(initialMigration)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// initialMigration is the fallback migration SQL
const initialMigration = `
CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    story_key TEXT NOT NULL,
    story_epic INTEGER NOT NULL,
    story_status TEXT NOT NULL,
    story_title TEXT,
    status TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT,
    duration_ms INTEGER DEFAULT 0,
    error TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS step_executions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    step_name TEXT NOT NULL,
    status TEXT NOT NULL,
    start_time TEXT,
    end_time TEXT,
    duration_ms INTEGER DEFAULT 0,
    attempt INTEGER DEFAULT 1,
    command TEXT,
    error TEXT,
    output_size INTEGER DEFAULT 0,
    FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS step_outputs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    step_execution_id TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    is_stderr BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (step_execution_id) REFERENCES step_executions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS step_averages (
    step_name TEXT PRIMARY KEY,
    avg_duration_ms INTEGER NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    last_updated TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_executions_story_key ON executions(story_key);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_start_time ON executions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_step_executions_execution_id ON step_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_step_executions_step_name ON step_executions(step_name);
CREATE INDEX IF NOT EXISTS idx_step_outputs_step_id ON step_outputs(step_execution_id);

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO schema_version (version) VALUES (1);
`

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// SaveExecution saves an execution and its steps to the database
func (s *SQLiteStorage) SaveExecution(ctx context.Context, exec *domain.Execution) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	execID := uuid.New().String()

	// Insert execution
	_, err = tx.ExecContext(ctx, `
		INSERT INTO executions (id, story_key, story_epic, story_status, story_title, status, start_time, end_time, duration_ms, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		execID,
		exec.Story.Key,
		exec.Story.Epic,
		string(exec.Story.Status),
		exec.Story.Title,
		string(exec.Status),
		exec.StartTime.Format(time.RFC3339),
		nullableTime(exec.EndTime),
		exec.Duration.Milliseconds(),
		nullableString(exec.Error),
	)
	if err != nil {
		return fmt.Errorf("failed to insert execution: %w", err)
	}

	// Insert steps
	for _, step := range exec.Steps {
		stepID := uuid.New().String()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO step_executions (id, execution_id, step_name, status, start_time, end_time, duration_ms, attempt, command, error, output_size)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			stepID,
			execID,
			string(step.Name),
			string(step.Status),
			nullableTime(step.StartTime),
			nullableTime(step.EndTime),
			step.Duration.Milliseconds(),
			step.Attempt,
			nullableString(step.Command),
			nullableString(step.Error),
			len(step.Output),
		)
		if err != nil {
			return fmt.Errorf("failed to insert step: %w", err)
		}

		// Insert step output lines (limit to prevent huge databases)
		maxLines := 1000
		outputLines := step.Output
		if len(outputLines) > maxLines {
			outputLines = outputLines[len(outputLines)-maxLines:]
		}

		// PERF-002 fix: Use bulk INSERT for step outputs
		if len(outputLines) > 0 {
			if err := s.bulkInsertStepOutputs(ctx, tx, stepID, outputLines); err != nil {
				return fmt.Errorf("failed to insert output lines: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetExecution retrieves an execution by ID (without output)
func (s *SQLiteStorage) GetExecution(ctx context.Context, id string) (*ExecutionRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, story_key, story_epic, story_status, story_title, status, start_time, end_time, duration_ms, error, created_at
		FROM executions WHERE id = ?
	`, id)

	rec, err := scanExecution(row)
	if err != nil {
		return nil, err
	}

	// Load steps (without output)
	steps, err := s.getSteps(ctx, id, false)
	if err != nil {
		return nil, err
	}
	rec.Steps = steps

	return rec, nil
}

// GetExecutionWithOutput retrieves an execution by ID with full output
func (s *SQLiteStorage) GetExecutionWithOutput(ctx context.Context, id string) (*ExecutionRecord, error) {
	rec, err := s.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load output for each step
	for _, step := range rec.Steps {
		output, err := s.GetStepOutput(ctx, step.ID)
		if err != nil {
			return nil, err
		}
		step.Output = output
	}

	return rec, nil
}

// ListExecutions retrieves executions matching the filter
// PERF-001 fix: Uses batch loading instead of N+1 queries
func (s *SQLiteStorage) ListExecutions(ctx context.Context, filter *ExecutionFilter) ([]*ExecutionRecord, error) {
	query := `
		SELECT id, story_key, story_epic, story_status, story_title, status, start_time, end_time, duration_ms, error, created_at
		FROM executions
	`
	where, args := buildWhereClause(filter)
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	var records []*ExecutionRecord
	var executionIDs []string
	for rows.Next() {
		rec, err := scanExecutionFromRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
		executionIDs = append(executionIDs, rec.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch load steps for all executions in one query (PERF-001 fix)
	if len(executionIDs) > 0 {
		stepsByExecution, err := s.getStepsBatch(ctx, executionIDs)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			rec.Steps = stepsByExecution[rec.ID]
		}
	}

	return records, nil
}

// CountExecutions returns the count of executions matching the filter
func (s *SQLiteStorage) CountExecutions(ctx context.Context, filter *ExecutionFilter) (int, error) {
	query := `SELECT COUNT(*) FROM executions`
	where, args := buildWhereClause(filter)
	if where != "" {
		query += " WHERE " + where
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// DeleteExecution deletes an execution and its related data
func (s *SQLiteStorage) DeleteExecution(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM executions WHERE id = ?", id)
	return err
}

// GetStepOutput retrieves output lines for a step
func (s *SQLiteStorage) GetStepOutput(ctx context.Context, stepID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT content FROM step_outputs
		WHERE step_execution_id = ?
		ORDER BY line_number
	`, stepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var output []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		output = append(output, line)
	}

	return output, rows.Err()
}

// GetStats returns aggregate statistics
func (s *SQLiteStorage) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		StepStats:        make(map[domain.StepName]*StepStats),
		ExecutionsByDay:  make(map[string]int),
		ExecutionsByEpic: make(map[int]int),
	}

	// Overall counts
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) as cancelled,
			COALESCE(AVG(duration_ms), 0) as avg_duration,
			COALESCE(SUM(duration_ms), 0) as total_duration
		FROM executions
	`).Scan(
		&stats.TotalExecutions,
		&stats.SuccessfulCount,
		&stats.FailedCount,
		&stats.CancelledCount,
		&stats.AvgDuration,
		&stats.TotalDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall stats: %w", err)
	}

	// Convert milliseconds to duration
	stats.AvgDuration = time.Duration(stats.AvgDuration) * time.Millisecond
	stats.TotalDuration = time.Duration(stats.TotalDuration) * time.Millisecond

	// Calculate success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessfulCount) / float64(stats.TotalExecutions) * 100
	}

	// Step stats
	stepRows, err := s.db.QueryContext(ctx, `
		SELECT
			step_name,
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) as successful,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) as skipped,
			COALESCE(AVG(CASE WHEN status = 'success' THEN duration_ms END), 0) as avg_duration,
			COALESCE(MIN(CASE WHEN status = 'success' THEN duration_ms END), 0) as min_duration,
			COALESCE(MAX(CASE WHEN status = 'success' THEN duration_ms END), 0) as max_duration
		FROM step_executions
		GROUP BY step_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get step stats: %w", err)
	}
	defer stepRows.Close()

	for stepRows.Next() {
		var ss StepStats
		var stepName string
		var avgMs, minMs, maxMs int64
		if err := stepRows.Scan(&stepName, &ss.TotalCount, &ss.SuccessCount, &ss.FailureCount, &ss.SkippedCount, &avgMs, &minMs, &maxMs); err != nil {
			return nil, err
		}
		ss.StepName = domain.StepName(stepName)
		ss.AvgDuration = time.Duration(avgMs) * time.Millisecond
		ss.MinDuration = time.Duration(minMs) * time.Millisecond
		ss.MaxDuration = time.Duration(maxMs) * time.Millisecond
		if ss.TotalCount > 0 {
			ss.SuccessRate = float64(ss.SuccessCount) / float64(ss.TotalCount) * 100
		}
		stats.StepStats[ss.StepName] = &ss
	}

	// Executions by day (last 30 days)
	dayRows, err := s.db.QueryContext(ctx, `
		SELECT date(created_at) as day, COUNT(*) as count
		FROM executions
		WHERE created_at >= datetime('now', '-30 days')
		GROUP BY day
		ORDER BY day DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by day: %w", err)
	}
	defer dayRows.Close()

	for dayRows.Next() {
		var day string
		var count int
		if err := dayRows.Scan(&day, &count); err != nil {
			return nil, err
		}
		stats.ExecutionsByDay[day] = count
	}

	// Executions by epic
	epicRows, err := s.db.QueryContext(ctx, `
		SELECT story_epic, COUNT(*) as count
		FROM executions
		GROUP BY story_epic
		ORDER BY story_epic
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by epic: %w", err)
	}
	defer epicRows.Close()

	for epicRows.Next() {
		var epic, count int
		if err := epicRows.Scan(&epic, &count); err != nil {
			return nil, err
		}
		stats.ExecutionsByEpic[epic] = count
	}

	// Recent executions (last 10)
	stats.RecentExecutions, err = s.GetRecentExecutions(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent executions: %w", err)
	}

	return stats, nil
}

// GetStepAverages returns historical averages for each step
func (s *SQLiteStorage) GetStepAverages(ctx context.Context) (map[domain.StepName]*StepAverage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT step_name, avg_duration_ms, success_count, failure_count, total_count, last_updated
		FROM step_averages
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	averages := make(map[domain.StepName]*StepAverage)
	for rows.Next() {
		var stepName string
		var avgMs int64
		var sa StepAverage
		var lastUpdated string

		if err := rows.Scan(&stepName, &avgMs, &sa.SuccessCount, &sa.FailureCount, &sa.TotalCount, &lastUpdated); err != nil {
			return nil, err
		}

		sa.StepName = domain.StepName(stepName)
		sa.AvgDuration = time.Duration(avgMs) * time.Millisecond
		sa.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
		averages[sa.StepName] = &sa
	}

	return averages, rows.Err()
}

// UpdateStepAverages recalculates and stores step averages
func (s *SQLiteStorage) UpdateStepAverages(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO step_averages (step_name, avg_duration_ms, success_count, failure_count, total_count, last_updated)
		SELECT
			step_name,
			COALESCE(AVG(CASE WHEN status = 'success' THEN duration_ms END), 0),
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),
			COUNT(*),
			datetime('now')
		FROM step_executions
		GROUP BY step_name
	`)
	return err
}

// GetRecentExecutions returns the most recent executions
func (s *SQLiteStorage) GetRecentExecutions(ctx context.Context, limit int) ([]*ExecutionRecord, error) {
	return s.ListExecutions(ctx, &ExecutionFilter{Limit: limit})
}

// GetExecutionsByStory returns all executions for a story
func (s *SQLiteStorage) GetExecutionsByStory(ctx context.Context, storyKey string) ([]*ExecutionRecord, error) {
	return s.ListExecutions(ctx, &ExecutionFilter{StoryKey: storyKey, Limit: 100})
}

// Helper functions

func (s *SQLiteStorage) getSteps(ctx context.Context, executionID string, includeOutput bool) ([]*StepRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, execution_id, step_name, status, start_time, end_time, duration_ms, attempt, command, error, output_size
		FROM step_executions
		WHERE execution_id = ?
		ORDER BY id
	`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*StepRecord
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return steps, rows.Err()
}

// getStepsBatch loads steps for multiple executions in one query (PERF-001 fix)
func (s *SQLiteStorage) getStepsBatch(ctx context.Context, executionIDs []string) (map[string][]*StepRecord, error) {
	if len(executionIDs) == 0 {
		return make(map[string][]*StepRecord), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(executionIDs))
	args := make([]any, len(executionIDs))
	for i, id := range executionIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, execution_id, step_name, status, start_time, end_time, duration_ms, attempt, command, error, output_size
		FROM step_executions
		WHERE execution_id IN (%s)
		ORDER BY execution_id, id
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch load steps: %w", err)
	}
	defer rows.Close()

	stepsByExecution := make(map[string][]*StepRecord)
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, err
		}
		stepsByExecution[step.ExecutionID] = append(stepsByExecution[step.ExecutionID], step)
	}

	return stepsByExecution, rows.Err()
}

func scanExecution(row *sql.Row) (*ExecutionRecord, error) {
	var rec ExecutionRecord
	var startTime, endTime, createdAt sql.NullString
	var durationMs int64
	var errStr sql.NullString
	var status, storyStatus string

	err := row.Scan(
		&rec.ID,
		&rec.StoryKey,
		&rec.StoryEpic,
		&storyStatus,
		&rec.StoryTitle,
		&status,
		&startTime,
		&endTime,
		&durationMs,
		&errStr,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("execution not found")
		}
		return nil, err
	}

	rec.Status = domain.ExecutionStatus(status)
	rec.StoryStatus = storyStatus
	rec.Duration = time.Duration(durationMs) * time.Millisecond

	if startTime.Valid {
		rec.StartTime, _ = time.Parse(time.RFC3339, startTime.String)
	}
	if endTime.Valid {
		rec.EndTime, _ = time.Parse(time.RFC3339, endTime.String)
	}
	if createdAt.Valid {
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if errStr.Valid {
		rec.Error = errStr.String
	}

	return &rec, nil
}

func scanExecutionFromRows(rows *sql.Rows) (*ExecutionRecord, error) {
	var rec ExecutionRecord
	var startTime, endTime, createdAt sql.NullString
	var durationMs int64
	var errStr sql.NullString
	var status, storyStatus string

	err := rows.Scan(
		&rec.ID,
		&rec.StoryKey,
		&rec.StoryEpic,
		&storyStatus,
		&rec.StoryTitle,
		&status,
		&startTime,
		&endTime,
		&durationMs,
		&errStr,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}

	rec.Status = domain.ExecutionStatus(status)
	rec.StoryStatus = storyStatus
	rec.Duration = time.Duration(durationMs) * time.Millisecond

	if startTime.Valid {
		rec.StartTime, _ = time.Parse(time.RFC3339, startTime.String)
	}
	if endTime.Valid {
		rec.EndTime, _ = time.Parse(time.RFC3339, endTime.String)
	}
	if createdAt.Valid {
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if errStr.Valid {
		rec.Error = errStr.String
	}

	return &rec, nil
}

func scanStep(rows *sql.Rows) (*StepRecord, error) {
	var step StepRecord
	var startTime, endTime sql.NullString
	var durationMs int64
	var errStr, cmd sql.NullString
	var stepName, status string

	err := rows.Scan(
		&step.ID,
		&step.ExecutionID,
		&stepName,
		&status,
		&startTime,
		&endTime,
		&durationMs,
		&step.Attempt,
		&cmd,
		&errStr,
		&step.OutputSize,
	)
	if err != nil {
		return nil, err
	}

	step.StepName = domain.StepName(stepName)
	step.Status = domain.StepStatus(status)
	step.Duration = time.Duration(durationMs) * time.Millisecond

	if startTime.Valid {
		step.StartTime, _ = time.Parse(time.RFC3339, startTime.String)
	}
	if endTime.Valid {
		step.EndTime, _ = time.Parse(time.RFC3339, endTime.String)
	}
	if cmd.Valid {
		step.Command = cmd.String
	}
	if errStr.Valid {
		step.Error = errStr.String
	}

	return &step, nil
}

// escapeLikeWildcards escapes SQL LIKE wildcards (% and _) in user input
// SEC-011: Prevents wildcard injection attacks in LIKE queries
func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // Escape backslash first
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func buildWhereClause(filter *ExecutionFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var conditions []string
	var args []any

	if filter.StoryKey != "" {
		// SEC-011: Escape LIKE wildcards to prevent injection
		conditions = append(conditions, "story_key LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escapeLikeWildcards(filter.StoryKey)+"%")
	}
	if filter.Epic != nil {
		conditions = append(conditions, "story_epic = ?")
		args = append(args, *filter.Epic)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.StartAfter != nil {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, filter.StartAfter.Format(time.RFC3339))
	}
	if filter.StartBefore != nil {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, filter.StartBefore.Format(time.RFC3339))
	}

	return strings.Join(conditions, " AND "), args
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// bulkInsertStepOutputs inserts multiple step output lines in batches (PERF-002 fix)
// SQLite has a limit on the number of variables (default 999), so we batch the inserts
func (s *SQLiteStorage) bulkInsertStepOutputs(ctx context.Context, tx *sql.Tx, stepID string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	// SQLite's default SQLITE_MAX_VARIABLE_NUMBER is 999
	// Each row uses 4 parameters, so max rows per batch is 249
	const maxRowsPerBatch = 200

	for batchStart := 0; batchStart < len(lines); batchStart += maxRowsPerBatch {
		batchEnd := batchStart + maxRowsPerBatch
		if batchEnd > len(lines) {
			batchEnd = len(lines)
		}
		batch := lines[batchStart:batchEnd]

		// Build the multi-value INSERT query
		var queryBuilder strings.Builder
		queryBuilder.WriteString("INSERT INTO step_outputs (step_execution_id, line_number, content, is_stderr) VALUES ")

		args := make([]any, 0, len(batch)*4)
		for i, line := range batch {
			if i > 0 {
				queryBuilder.WriteString(",")
			}
			queryBuilder.WriteString("(?,?,?,?)")
			args = append(args, stepID, batchStart+i, line, false)
		}

		_, err := tx.ExecContext(ctx, queryBuilder.String(), args...)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetDatabasePath returns the default database path
func GetDatabasePath(dataDir string) string {
	return filepath.Join(dataDir, "bmad.db")
}
