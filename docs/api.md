# REST API Reference

BMAD Automate provides a REST API for programmatic control and integration with external tools. The API includes WebSocket support for real-time updates.

## Getting Started

### Enabling the API Server

Start BMAD with the API server enabled:

```bash
# Via command line
bmad --api --port 8080

# Via make
make run-api
```

The API server will start on `http://localhost:8080`.

### Base URL

All API endpoints are prefixed with `/api` except for the health check:

```
http://localhost:8080/api/...
```

### Response Format

All responses are JSON formatted:

```json
{
  "data": { ... },
  "error": "error message (if applicable)"
}
```

## Authentication

Currently, the API does not require authentication. CORS is enabled for all origins.

## Endpoints

### Health Check

Check if the API server is running.

```http
GET /health
```

**Response**

```json
{
  "status": "ok",
  "time": "2024-01-15T10:30:00Z"
}
```

---

## Stories

### List Stories

Get all stories from `sprint-status.yaml`.

```http
GET /api/stories
```

**Query Parameters**

| Parameter | Type    | Description                                                                     |
| --------- | ------- | ------------------------------------------------------------------------------- |
| `epic`    | integer | Filter by epic number                                                           |
| `status`  | string  | Filter by status (`in-progress`, `ready-for-dev`, `backlog`, `done`, `blocked`) |

**Example Request**

```bash
curl "http://localhost:8080/api/stories?epic=3&status=ready-for-dev"
```

**Response**

```json
{
  "stories": [
    {
      "Key": "3-1-user-auth",
      "Epic": 3,
      "Status": "ready-for-dev",
      "Title": "User Authentication",
      "FilePath": "/project/stories/3-1-user-auth.md",
      "FileExists": true
    }
  ],
  "count": 1
}
```

### Get Story

Get a specific story by key.

```http
GET /api/stories/{key}
```

**Example Request**

```bash
curl "http://localhost:8080/api/stories/3-1-user-auth"
```

**Response**

```json
{
  "Key": "3-1-user-auth",
  "Epic": 3,
  "Status": "ready-for-dev",
  "Title": "User Authentication",
  "FilePath": "/project/stories/3-1-user-auth.md",
  "FileExists": true
}
```

**Error Response (404)**

```json
{
  "error": "story not found"
}
```

### Refresh Stories

Reload stories from `sprint-status.yaml`.

```http
POST /api/stories/refresh
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/stories/refresh"
```

**Response**

```json
{
  "stories": [...],
  "count": 15
}
```

---

## Queue Management

### Get Queue

Get the current queue status.

```http
GET /api/queue
```

**Example Request**

```bash
curl "http://localhost:8080/api/queue"
```

**Response**

```json
{
  "items": [
    {
      "story": {
        "Key": "3-1-user-auth",
        "Epic": 3,
        "Status": "ready-for-dev",
        "Title": "User Authentication"
      },
      "status": "pending",
      "position": 0,
      "added_at": "2024-01-15T10:30:00Z"
    }
  ],
  "status": "idle",
  "current": -1,
  "total": 1,
  "pending": 1,
  "eta": 1200.5
}
```

**Queue Status Values**

| Status      | Description          |
| ----------- | -------------------- |
| `idle`      | Queue is not running |
| `running`   | Queue is executing   |
| `paused`    | Queue is paused      |
| `completed` | All items processed  |

### Add Stories to Queue

Add multiple stories to the queue.

```http
POST /api/queue/add
Content-Type: application/json
```

**Request Body**

```json
{
  "keys": ["3-1-user-auth", "3-2-password-reset"]
}
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/queue/add" \
  -H "Content-Type: application/json" \
  -d '{"keys": ["3-1-user-auth", "3-2-password-reset"]}'
```

**Response**

```json
{
  "added": 2,
  "queue": 2
}
```

### Add Single Story to Queue

Add a single story to the queue by key.

```http
POST /api/queue/add/{key}
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/queue/add/3-1-user-auth"
```

**Response**

```json
{
  "added": 1,
  "queue": 1
}
```

### Remove from Queue

Remove a story from the queue.

```http
DELETE /api/queue/{key}
```

**Example Request**

```bash
curl -X DELETE "http://localhost:8080/api/queue/3-1-user-auth"
```

**Response**

```json
{
  "status": "removed"
}
```

### Clear Queue

Remove all pending items from the queue.

```http
POST /api/queue/clear
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/queue/clear"
```

**Response**

```json
{
  "status": "cleared"
}
```

### Reorder Queue

Move an item up or down in the queue.

```http
POST /api/queue/reorder
Content-Type: application/json
```

**Request Body**

```json
{
  "index": 0,
  "direction": "down"
}
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/queue/reorder" \
  -H "Content-Type: application/json" \
  -d '{"index": 0, "direction": "down"}'
```

**Response**

```json
{
  "status": "reordered"
}
```

---

## Execution Control

### Get Execution Status

Get the current execution state.

```http
GET /api/execution
```

**Example Request**

```bash
curl "http://localhost:8080/api/execution"
```

**Response (Running)**

```json
{
  "running": true,
  "status": "running",
  "story": {
    "Key": "3-1-user-auth",
    "Epic": 3,
    "Title": "User Authentication"
  },
  "current": 1,
  "steps": [
    {
      "name": "create-story",
      "status": "skipped",
      "duration": 0,
      "attempt": 0,
      "error": ""
    },
    {
      "name": "dev-story",
      "status": "running",
      "duration": 45.5,
      "attempt": 1,
      "error": ""
    },
    {
      "name": "code-review",
      "status": "pending",
      "duration": 0,
      "attempt": 0,
      "error": ""
    },
    {
      "name": "git-commit",
      "status": "pending",
      "duration": 0,
      "attempt": 0,
      "error": ""
    }
  ],
  "duration": 47.2,
  "progress": 25
}
```

**Response (Not Running)**

```json
{
  "running": false
}
```

### Start Queue Execution

Start processing the queue.

```http
POST /api/execution/start
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/start"
```

**Response**

```json
{
  "status": "started"
}
```

**Error Responses**

```json
{"error": "no items in queue"}
{"error": "execution already running"}
```

### Start Single Story Execution

Start execution of a specific story.

```http
POST /api/execution/start/{key}
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/start/3-1-user-auth"
```

**Response**

```json
{
  "status": "started"
}
```

### Pause Execution

Pause the current execution.

```http
POST /api/execution/pause
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/pause"
```

**Response**

```json
{
  "status": "paused"
}
```

### Resume Execution

Resume a paused execution.

```http
POST /api/execution/resume
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/resume"
```

**Response**

```json
{
  "status": "resumed"
}
```

### Cancel Execution

Cancel the current execution.

```http
POST /api/execution/cancel
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/cancel"
```

**Response**

```json
{
  "status": "cancelled"
}
```

### Skip Current Step

Skip the currently running step.

```http
POST /api/execution/skip
```

**Example Request**

```bash
curl -X POST "http://localhost:8080/api/execution/skip"
```

**Response**

```json
{
  "status": "skipping"
}
```

---

## History

### List Execution History

Get past executions.

```http
GET /api/history
```

**Query Parameters**

| Parameter | Type    | Description                | Default |
| --------- | ------- | -------------------------- | ------- |
| `limit`   | integer | Maximum records to return  | 50      |
| `story`   | string  | Filter by story key        |         |
| `epic`    | integer | Filter by epic number      |         |
| `status`  | string  | Filter by execution status |         |

**Example Request**

```bash
curl "http://localhost:8080/api/history?limit=10&status=completed"
```

**Response**

```json
{
  "executions": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "story_key": "3-1-user-auth",
      "story_epic": 3,
      "status": "completed",
      "start_time": "2024-01-15T10:30:00Z",
      "duration": 245.5,
      "error": ""
    }
  ],
  "count": 1,
  "total": 25
}
```

### Get Execution Details

Get detailed information about a specific execution, including step output.

```http
GET /api/history/{id}
```

**Example Request**

```bash
curl "http://localhost:8080/api/history/550e8400-e29b-41d4-a716-446655440000"
```

**Response**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "story_key": "3-1-user-auth",
  "story_epic": 3,
  "status": "completed",
  "start_time": "2024-01-15T10:30:00Z",
  "end_time": "2024-01-15T10:34:05Z",
  "duration": 245.5,
  "error": "",
  "steps": [
    {
      "name": "create-story",
      "status": "skipped",
      "duration": 0,
      "attempt": 0,
      "command": "",
      "error": "",
      "output": []
    },
    {
      "name": "dev-story",
      "status": "success",
      "duration": 180.2,
      "attempt": 1,
      "command": "claude --dangerously-skip-permissions -p \"...\"",
      "error": "",
      "output": ["Starting implementation...", "Creating user model...", "..."]
    }
  ]
}
```

---

## Statistics

### Get Statistics

Get execution statistics and trends.

```http
GET /api/stats
```

**Example Request**

```bash
curl "http://localhost:8080/api/stats"
```

**Response**

```json
{
  "total_executions": 50,
  "successful": 42,
  "failed": 5,
  "cancelled": 3,
  "success_rate": 84.0,
  "avg_duration": 312.5,
  "total_duration": 15625.0,
  "step_stats": {
    "create-story": {
      "total": 50,
      "success": 10,
      "failure": 0,
      "skipped": 40,
      "success_rate": 100.0,
      "avg_duration": 15.2,
      "min_duration": 10.1,
      "max_duration": 25.5
    },
    "dev-story": {
      "total": 50,
      "success": 45,
      "failure": 5,
      "skipped": 0,
      "success_rate": 90.0,
      "avg_duration": 180.5,
      "min_duration": 60.2,
      "max_duration": 450.8
    },
    "code-review": {
      "total": 45,
      "success": 42,
      "failure": 3,
      "skipped": 0,
      "success_rate": 93.3,
      "avg_duration": 90.2,
      "min_duration": 30.5,
      "max_duration": 200.1
    },
    "git-commit": {
      "total": 42,
      "success": 42,
      "failure": 0,
      "skipped": 0,
      "success_rate": 100.0,
      "avg_duration": 25.5,
      "min_duration": 10.2,
      "max_duration": 45.8
    }
  },
  "executions_by_day": {
    "2024-01-14": 8,
    "2024-01-15": 12
  },
  "executions_by_epic": {
    "1": 10,
    "2": 15,
    "3": 25
  }
}
```

---

## Configuration

### Get Configuration

Get the current configuration.

```http
GET /api/config
```

**Example Request**

```bash
curl "http://localhost:8080/api/config"
```

**Response**

```json
{
  "working_dir": "/Users/user/project",
  "sprint_status": "/Users/user/project/_bmad-output/implementation-artifacts/sprint-status.yaml",
  "story_dir": "/Users/user/project/_bmad-output/implementation-artifacts",
  "timeout": 600,
  "retries": 1,
  "theme": "catppuccin",
  "sound_enabled": false,
  "notifications": true
}
```

---

## WebSocket

Connect to receive real-time updates.

```http
GET /api/ws
```

### Connection

```javascript
const ws = new WebSocket("ws://localhost:8080/api/ws");

ws.onopen = () => {
  console.log("Connected to BMAD");
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log("Received:", message);
};
```

### Message Types

All WebSocket messages follow this format:

```json
{
  "type": "message_type",
  "data": { ... },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Message Types**

| Type                  | Description               |
| --------------------- | ------------------------- |
| `execution_started`   | Execution has begun       |
| `step_started`        | A step has started        |
| `step_output`         | New output line from step |
| `step_completed`      | A step has finished       |
| `execution_completed` | Execution has finished    |
| `queue_updated`       | Queue state changed       |
| `stories_refreshed`   | Stories were reloaded     |

**Example Messages**

```json
// execution_started
{
  "type": "execution_started",
  "data": {
    "story_key": "3-1-user-auth",
    "story_title": "User Authentication"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}

// step_output
{
  "type": "step_output",
  "data": {
    "step": "dev-story",
    "line": "Creating user model...",
    "is_stderr": false
  },
  "timestamp": "2024-01-15T10:30:15Z"
}

// execution_completed
{
  "type": "execution_completed",
  "data": {
    "story_key": "3-1-user-auth",
    "status": "completed",
    "duration": 245.5
  },
  "timestamp": "2024-01-15T10:34:05Z"
}
```

---

## Error Handling

### Error Response Format

All errors return JSON with an `error` field:

```json
{
  "error": "description of the error"
}
```

### HTTP Status Codes

| Code | Description                                       |
| ---- | ------------------------------------------------- |
| 200  | Success                                           |
| 400  | Bad Request - Invalid parameters                  |
| 404  | Not Found - Resource doesn't exist                |
| 409  | Conflict - Operation not allowed in current state |
| 500  | Internal Server Error                             |
| 503  | Service Unavailable - Storage not available       |

### Common Errors

```json
// Story not found
{"error": "story not found"}

// Execution already running
{"error": "execution already running"}

// No items in queue
{"error": "no items in queue"}

// Storage not available
{"error": "storage not available"}
```

---

## Code Examples

### Python Client

```python
import requests
import json

BASE_URL = "http://localhost:8080/api"

class BMADClient:
    def __init__(self, base_url=BASE_URL):
        self.base_url = base_url

    def get_stories(self, epic=None, status=None):
        params = {}
        if epic:
            params['epic'] = epic
        if status:
            params['status'] = status
        response = requests.get(f"{self.base_url}/stories", params=params)
        return response.json()

    def add_to_queue(self, keys):
        response = requests.post(
            f"{self.base_url}/queue/add",
            json={"keys": keys}
        )
        return response.json()

    def start_execution(self):
        response = requests.post(f"{self.base_url}/execution/start")
        return response.json()

    def get_status(self):
        response = requests.get(f"{self.base_url}/execution")
        return response.json()

# Usage
client = BMADClient()
stories = client.get_stories(status="ready-for-dev")
client.add_to_queue([s['Key'] for s in stories['stories']])
client.start_execution()
```

### JavaScript/Node.js Client

```javascript
const WebSocket = require("ws");

class BMADClient {
  constructor(baseUrl = "http://localhost:8080") {
    this.baseUrl = baseUrl;
  }

  async getStories(filters = {}) {
    const params = new URLSearchParams(filters);
    const response = await fetch(`${this.baseUrl}/api/stories?${params}`);
    return response.json();
  }

  async addToQueue(keys) {
    const response = await fetch(`${this.baseUrl}/api/queue/add`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ keys }),
    });
    return response.json();
  }

  connectWebSocket(onMessage) {
    const ws = new WebSocket(
      `ws://${this.baseUrl.replace("http://", "")}/api/ws`,
    );
    ws.on("message", (data) => onMessage(JSON.parse(data)));
    return ws;
  }
}

// Usage
const client = new BMADClient();
const stories = await client.getStories({ status: "ready-for-dev" });
await client.addToQueue(stories.stories.map((s) => s.Key));

client.connectWebSocket((msg) => {
  console.log(`[${msg.type}]`, msg.data);
});
```

### cURL Examples

```bash
# List ready stories
curl "http://localhost:8080/api/stories?status=ready-for-dev"

# Add stories to queue
curl -X POST "http://localhost:8080/api/queue/add" \
  -H "Content-Type: application/json" \
  -d '{"keys": ["3-1-user-auth", "3-2-password-reset"]}'

# Start execution
curl -X POST "http://localhost:8080/api/execution/start"

# Monitor status
watch -n 2 'curl -s "http://localhost:8080/api/execution" | jq'

# Get statistics
curl "http://localhost:8080/api/stats" | jq
```
