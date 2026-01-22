# Runqy

A task queueing service built on [asynq](https://github.com/hibiken/asynq) for managing distributed task processing with Redis-backed queues.

![Asynq Architecture](https://user-images.githubusercontent.com/11155743/116358505-656f5f80-a806-11eb-9c16-94e49dab0f99.jpg)

## Features

- REST API for enqueueing and monitoring tasks
- Redis-backed persistent queue with retry support
- PostgreSQL storage for queue worker configurations
- YAML-based queue worker definitions with schema validation
- Built-in web dashboard for monitoring (asynqmon)
- Swagger API documentation
- Hot-reload of queue configurations (file watching or git polling)

## Requirements

- Go 1.24+
- Redis
- PostgreSQL

## Quick Start

### 1. Set up environment

Copy the sample environment file and configure your settings:

```shell
cp .env.secret.sample .env.secret
```

### 2. Run with Docker (recommended)

```shell
# Linux / macOS
docker run --rm -it --init \
  -p 3000:3000 \
  --env-file .env.secret \
  -v "$(pwd)/app:/app" \
  -w /app \
  -v go-mod-cache:/go/pkg/mod \
  -v go-build-cache:/root/.cache/go-build \
  golang:1.25.3 \
  bash -c "go mod download && exec go run ."

# Windows PowerShell
docker run --rm -it --init `
  -p 3000:3000 `
  --env-file .env.secret `
  -v "${PWD}\app:/app" `
  -w /app `
  -v go-mod-cache:/go/pkg/mod `
  -v go-build-cache:/root/.cache/go-build `
  golang:1.25.3 `
  bash -c "go mod download && exec go run ."
```

### 3. Or run directly

```shell
cd app
go run .
```

### 4. Build Docker image

```shell
docker build -t runqy .
docker run -it --rm --name runqy -p 3000:3000 --env-file .env.secret runqy
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | 3000 | HTTP server port |
| `REDIS_HOST` | Yes | localhost | Redis hostname |
| `REDIS_PORT` | No | 6379 | Redis port |
| `REDIS_PASSWORD` | Yes | - | Redis password |
| `REDIS_TLS` | No | false | Enable TLS for Redis |
| `DATABASE_HOST` | Yes | localhost | PostgreSQL hostname |
| `DATABASE_PORT` | No | 5432 | PostgreSQL port |
| `DATABASE_USER` | No | postgres | PostgreSQL user |
| `DATABASE_PASSWORD` | Yes | - | PostgreSQL password |
| `DATABASE_DBNAME` | No | runqy_dev | PostgreSQL database |
| `ASYNQ_API_KEY` | Yes | - | API key for authenticated endpoints |
| `QUEUE_WORKERS_DIR` | No | ../deployment | Path to queue worker YAML configs |

## API Usage

### Enqueue a Task

```
POST /queue/add
Authorization: Bearer <ASYNQ_API_KEY>
Content-Type: application/json
```

```json
{
  "queue": "my_queue",
  "timeout": 300,
  "data": {
    "prompt": "Hello world",
    "width": 1024,
    "height": 1024
  }
}
```

Response:
```json
{
  "info": {
    "id": "task-uuid",
    "queue": "my_queue",
    "state": "pending"
  }
}
```

### Get Task Status

```
GET /queue/{uuid}/{queue_name}
```

Response:
```json
{
  "info": {
    "id": "task-uuid",
    "state": "completed",
    "result": "..."
  }
}
```

## Queue Worker Configuration

Define queue workers in YAML files under the `deployment/` directory:

```yaml
queues:
  my_queue:
    priority: 5
    deployment:
      git_url: "https://github.com/org/worker.git"
      branch: "main"
      startup_cmd: "python main.py"
      startup_timeout_secs: 60
    input:
      - name: "prompt"
        type: ["string"]
      - name: "width"
        type: ["int"]
```

## Monitoring

### Web Dashboard

Visit http://localhost:3000/monitoring/

![Asynq Dashboard](https://github.com/hibiken/asynq/blob/master/docs/assets/dash.gif)

## CLI Commands

Runqy includes a comprehensive CLI for managing queues, tasks, workers, and configurations without needing to use the REST API or external tools.

### Build the CLI

```shell
cd app
go build -o runqy .
```

### Command Overview

| Command | Description |
|---------|-------------|
| `runqy` | Start the HTTP server (default, same as `runqy serve`) |
| `runqy serve` | Start the HTTP server |
| `runqy queue` | Queue management commands |
| `runqy task` | Task management commands |
| `runqy worker` | Worker management commands |
| `runqy config` | Configuration management commands |
| `runqy login` | Save server credentials for remote mode |
| `runqy logout` | Remove saved credentials |
| `runqy auth` | Authentication management (status, list, switch) |

### Server Commands

#### Start the Server

```shell
# Start with defaults
runqy serve

# Or simply (serve is the default command)
runqy

# Start with custom config directory
runqy serve --config ./my-deployment

# Start with git-based config and auto-reload
runqy serve --config-repo https://github.com/org/configs.git --watch

# All serve options
runqy serve --help
```

**Serve Flags:**

| Flag | Description |
|------|-------------|
| `--config` | Path to queue workers config directory |
| `--watch` | Enable file/git watching for config auto-reload |
| `--config-repo` | GitHub repo URL for configs |
| `--config-branch` | Git branch (default: main) |
| `--config-path` | Path within repo to YAML files |
| `--clone-dir` | Directory to clone repo into |
| `--watch-interval` | Git polling interval in seconds |

### Queue Commands

```shell
# List all queues with statistics
runqy queue list

# Output:
# QUEUE              PENDING  ACTIVE  SCHEDULED  RETRY  ARCHIVED  COMPLETED  PAUSED
# inference_high     5        2       0          1      0         150        no
# inference_low      12       0       3          0      0         89         no

# Show detailed queue information
runqy queue inspect inference_high

# Pause a queue (stops processing new tasks)
runqy queue pause inference_high

# Resume a paused queue
runqy queue unpause inference_high
```

### Task Commands

#### Enqueue a Task

```shell
# Enqueue a task with JSON payload
runqy task enqueue --queue inference_high --payload '{"prompt":"Hello world","width":1024}'

# Short flags
runqy task enqueue -q inference_high -p '{"msg":"test"}'

# With custom timeout (seconds)
runqy task enqueue -q inference_high -p '{"data":"value"}' --timeout 300
```

#### List Tasks

```shell
# List pending tasks in a queue
runqy task list inference_high

# List tasks by state
runqy task list inference_high --state pending
runqy task list inference_high --state active
runqy task list inference_high --state scheduled
runqy task list inference_high --state retry
runqy task list inference_high --state archived
runqy task list inference_high --state completed

# Limit number of results
runqy task list inference_high --state pending --limit 20
```

#### Get Task Details

```shell
# Get detailed info about a specific task
runqy task get inference_high abc123-task-id

# Output:
# Task ID:     abc123-task-id
# Type:        task
# Queue:       inference_high
# State:       completed
# Max Retry:   3
# Retried:     0
# Timeout:     10m0s
# ...
```

#### Cancel and Delete Tasks

```shell
# Cancel a running task
runqy task cancel abc123-task-id

# Delete a task from a queue
runqy task delete inference_high abc123-task-id
```

### Worker Commands

```shell
# List all registered workers
runqy worker list

# Output:
# WORKER_ID                              STATUS  QUEUES          CONCURRENCY  LAST_BEAT  STALE
# worker-abc123-def456                   ready   inference_high  1            5s         no
# worker-xyz789-uvw012                   ready   inference_low   1            3s         no

# Show detailed worker information
runqy worker info worker-abc123-def456

# Output:
# Worker ID:   worker-abc123-def456
# Status:      ready
# Queues:      inference_high
# Concurrency: 1
# Started At:  2024-01-15 10:30:00
# Last Beat:   2024-01-15 10:35:45 (5s ago)
```

### Config Commands

#### List Queue Configurations

```shell
# List all queue configurations from PostgreSQL
runqy config list

# Output:
# NAME              PRIORITY  PROVIDER  MODE          GIT_URL
# inference_high    10        worker    long_running  https://github.com/org/worker.git
# inference_low     5         worker    long_running  https://github.com/org/worker.git
# simple_default    1         worker    one_shot      https://github.com/org/simple.git
```

#### Reload Configurations

```shell
# Reload configs from default directory (QUEUE_WORKERS_DIR)
runqy config reload

# Reload from a specific directory
runqy config reload --dir ./my-deployment

# Output:
# Reloaded 3 queue configuration(s):
#   - inference_high
#   - inference_low
#   - simple_default
```

#### Validate Configuration Files

```shell
# Validate YAML files without loading into database
runqy config validate

# Validate from a specific directory
runqy config validate --dir ./my-deployment

# Output:
# Validated 2 YAML file(s) from ./my-deployment
#   - inference_high (priority=10)
#   - inference_low (priority=5)
#
# Total: 2 queue configuration(s)
# Validation successful!
```

### Global Flags

These flags are available for all commands:

| Flag | Description |
|------|-------------|
| `-s, --server` | Remote server URL for CLI-over-HTTP mode |
| `-k, --api-key` | API key for authentication (or set ASYNQ_API_KEY env var) |
| `--redis-uri` | Redis URI (overrides REDIS_HOST/REDIS_PORT) |
| `-v, --version` | Print version information |
| `-h, --help` | Help for the command |

### Remote Mode (CLI over HTTP)

The CLI can operate in two modes:

1. **Local mode** (default): Connects directly to Redis/PostgreSQL
2. **Remote mode**: Connects to a runqy server via HTTP API

Use remote mode when you want to manage a runqy server from a different machine (e.g., from your laptop to a production server).

#### Usage

```shell
# Remote mode - specify server URL and API key
runqy --server https://runqy.example.com:3000 --api-key YOUR_API_KEY queue list

# Short flags
runqy -s https://runqy.example.com:3000 -k YOUR_API_KEY queue list

# API key can also be set via environment variable
export ASYNQ_API_KEY=YOUR_API_KEY
runqy -s https://runqy.example.com:3000 queue list
```

#### Remote Mode Examples

```shell
# List queues on remote server
runqy -s https://server:3000 -k API_KEY queue list

# Enqueue a task on remote server
runqy -s https://server:3000 -k API_KEY task enqueue -q inference_high -p '{"msg":"hello"}'

# List workers on remote server
runqy -s https://server:3000 -k API_KEY worker list

# List configs on remote server
runqy -s https://server:3000 -k API_KEY config list

# Trigger config reload on remote server
runqy -s https://server:3000 -k API_KEY config reload
```

#### Remote Mode Limitations

| Command | Remote Support | Notes |
|---------|---------------|-------|
| `queue list/inspect/pause/unpause` | Yes | Full support |
| `task enqueue/list/get/cancel/delete` | Yes | Full support |
| `worker list/info` | Yes | Full support |
| `config list/reload` | Yes | Full support |
| `config validate` | No | Local-only (validates local YAML files) |
| `serve` | No | Server command, not applicable |

### Authentication Persistence

Save server credentials so you don't need to specify `--server` and `--api-key` for every command.

#### Login and Save Credentials

```shell
# Save credentials for a server (saved as "default" profile)
runqy login -s https://production.example.com:3000 -k prod-api-key

# Save with a custom profile name
runqy login -s https://staging.example.com:3000 -k staging-key --name staging

# API key can be prompted interactively
runqy login -s https://server:3000
# API Key: <enter key>
```

Credentials are stored in `~/.runqy/credentials.json` with restricted permissions.

#### Using Saved Credentials

After logging in, commands work without flags:

```shell
# Before (verbose)
runqy --server https://server:3000 --api-key KEY queue list

# After login (simple)
runqy queue list
runqy task enqueue -q myqueue -p '{"msg":"hello"}'
runqy worker list
```

#### Manage Multiple Servers

```shell
# List all saved servers
runqy auth list
# NAME     URL                                    CURRENT
# default  https://production.example.com:3000   *
# staging  https://staging.example.com:3000

# Show current connection
runqy auth status
# Current server: default
# URL: https://production.example.com:3000
# API Key: prod...key

# Switch to different server
runqy auth switch staging
# Switched to "staging"
```

#### Logout

```shell
# Remove current profile
runqy logout

# Remove specific profile
runqy logout --name staging

# Remove all saved credentials
runqy logout --all
```

#### Credential Priority

Credentials are resolved in this order (highest to lowest):

1. Command-line flags (`--server`, `--api-key`)
2. Environment variables (`RUNQY_SERVER`, `ASYNQ_API_KEY`)
3. Saved credentials (`~/.runqy/credentials.json`)
4. Local mode (direct Redis/PostgreSQL access)

### Shell Completion

Generate shell completion scripts for your shell:

```shell
# Bash
runqy completion bash > /etc/bash_completion.d/runqy

# Zsh
runqy completion zsh > "${fpath[1]}/_runqy"

# Fish
runqy completion fish > ~/.config/fish/completions/runqy.fish

# PowerShell
runqy completion powershell > runqy.ps1
```

## API Documentation

Swagger UI available at: http://localhost:3000/swagger/index.html

## Development

### Run tests

```shell
cd app
go test ./test/...
```

### Generate Swagger docs

```shell
cd app
swag init
```

### Load testing

```shell
cd app/vegeta
vegeta attack -targets=targets.txt -rate=15 -duration=1s > results.bin
cat results.bin | vegeta report -type=text
```

## License

MIT License - see [LICENSE](LICENSE) for details.
