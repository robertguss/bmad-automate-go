-- 001_initial.sql
-- Initial database schema for bmad-automate-go

-- Executions table: stores overall execution records
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

-- Step executions table: stores individual step records
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

-- Step output table: stores output lines (separate for performance)
CREATE TABLE IF NOT EXISTS step_outputs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    step_execution_id TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    is_stderr BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (step_execution_id) REFERENCES step_executions(id) ON DELETE CASCADE
);

-- Step averages table: stores historical averages for ETA calculation
CREATE TABLE IF NOT EXISTS step_averages (
    step_name TEXT PRIMARY KEY,
    avg_duration_ms INTEGER NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    last_updated TEXT NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_executions_story_key ON executions(story_key);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_start_time ON executions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_step_executions_execution_id ON step_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_step_executions_step_name ON step_executions(step_name);
CREATE INDEX IF NOT EXISTS idx_step_outputs_step_id ON step_outputs(step_execution_id);

-- Schema version table for migrations
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Insert initial version
INSERT OR IGNORE INTO schema_version (version) VALUES (1);
