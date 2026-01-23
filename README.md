<p align="center">
  <img src="assets/logo.svg" alt="runqy logo" width="80" height="80">
</p>

<h1 align="center">runqy</h1>

<p align="center">
  A distributed task queue system with server-driven bootstrap architecture.
  <br>
  <a href="https://docs.runqy.com"><strong>Documentation</strong></a> · <a href="https://runqy.com"><strong>Website</strong></a>
</p>

## Features

- REST API for enqueueing and monitoring tasks
- Redis-backed persistent queue with retry support
- PostgreSQL storage for queue worker configurations
- YAML-based queue worker definitions with schema validation
- Built-in web dashboard for monitoring (asynqmon)
- Swagger API documentation
- Hot-reload of queue configurations (file watching or git polling)

## Installation

### Quick Install (Recommended)

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/publikey/runqy/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
iwr https://raw.githubusercontent.com/publikey/runqy/main/install.ps1 -useb | iex
```

### Docker

```bash
docker pull ghcr.io/publikey/runqy:latest
```

### Docker Compose Quickstart

Run the full stack without cloning the repo:

```bash
# Download and start
curl -O https://raw.githubusercontent.com/Publikey/runqy/main/docker-compose.quickstart.yml
docker-compose -f docker-compose.quickstart.yml up -d

# Access dashboard
open http://localhost:3000/monitoring/
```

### From Source

Requires Go 1.24+:
```bash
git clone https://github.com/Publikey/runqy.git
cd runqy/app
go build -o runqy .
```

## Requirements

- Redis
- PostgreSQL (only for production - SQLite is embedded for development)

## Quick Start

**Step 1: Install runqy and runqy-worker**

See [Installation](#installation) above, or use Docker Compose Quickstart (no source code required):
```bash
curl -O https://raw.githubusercontent.com/Publikey/runqy/main/docker-compose.quickstart.yml
docker-compose -f docker-compose.quickstart.yml up -d
```

Or clone the repo for full development setup:
```bash
git clone https://github.com/Publikey/runqy.git
cd runqy
docker-compose up -d
```

**Step 2: Start Redis** (skip if using Docker Compose)
```bash
docker run -d --name redis -p 6379:6379 redis:alpine
```

**Step 3: Start the server** (skip if using Docker Compose)

Linux/Mac:
```bash
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=""
export RUNQY_API_KEY=dev-api-key
runqy serve --sqlite
```

Windows (PowerShell):
```powershell
$env:REDIS_HOST = "localhost"
$env:REDIS_PORT = "6379"
$env:REDIS_PASSWORD = ""
$env:RUNQY_API_KEY = "dev-api-key"
runqy serve --sqlite
```

**Step 4: Deploy the example queues** (in another terminal, skip if using Docker Compose)

Linux/Mac:
```bash
# Download example config
curl -fsSL https://raw.githubusercontent.com/Publikey/runqy/main/examples/quickstart.yaml -o quickstart.yaml

runqy login -s http://localhost:3000 -k dev-api-key
runqy config create -f quickstart.yaml
```

Windows (PowerShell):
```powershell
# Download example config
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/Publikey/runqy/main/examples/quickstart.yaml" -OutFile "quickstart.yaml"

runqy login -s http://localhost:3000 -k dev-api-key
runqy config create -f quickstart.yaml
```

This deploys two example queues:
- `quickstart-oneshot` - spawns a new Python process per task
- `quickstart-longrunning` - keeps Python process alive between tasks

**Step 5: Start a worker** (in another terminal, skip if using Docker Compose)

Install `runqy-worker` using one of these methods:

**Option A: Quick Install**
```bash
curl -fsSL https://raw.githubusercontent.com/publikey/runqy-worker/main/install.sh | sh
```

**Option B: Binary Download**
```bash
curl -LO https://github.com/publikey/runqy-worker/releases/latest/download/runqy-worker_latest_linux_amd64.tar.gz
tar -xzf runqy-worker_latest_linux_amd64.tar.gz
```

**Option C: Docker**
```bash
docker pull ghcr.io/publikey/runqy-worker:latest
```

Then download the example config and run:
```bash
# Download example config
curl -fsSL https://raw.githubusercontent.com/publikey/runqy-worker/main/config.yml.example -o config.yml

# Start worker (binary)
runqy-worker -config config.yml

# Or with Docker
docker run -v $(pwd)/config.yml:/app/config.yml ghcr.io/publikey/runqy-worker:latest
```

The example config is pre-configured for the quickstart with both queues:

```yaml
worker:
  queues:
    - "quickstart-oneshot_default"
    - "quickstart-longrunning_default"
```

The worker registers with the server, clones the example task code, and starts processing.

**Step 6: Enqueue a task**

Linux/Mac:
```bash
curl -X POST http://localhost:3000/queue/add \
  -H "Authorization: Bearer dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "quickstart-oneshot_default",
    "timeout": 60,
    "data": {"operation": "uppercase", "data": "hello world"}
  }'
```

Windows (PowerShell):
```powershell
curl.exe -X POST http://localhost:3000/queue/add `
  -H "Authorization: Bearer dev-api-key" `
  -H "Content-Type: application/json" `
  -d '{\"queue\": \"quickstart-oneshot_default\", \"timeout\": 60, \"data\": {\"operation\": \"uppercase\", \"data\": \"hello world\"}}'
```

Response:
```json
{"info": {"id": "abc123...", "state": "pending", "queue": "quickstart-oneshot_default", ...}, "data": {...}}
```

Use the `id` from the response in the next step.

To try long-running mode, just enqueue to `quickstart-longrunning_default` — the worker already listens on both queues.

**Step 7: Check the result**

Linux/Mac:
```bash
curl http://localhost:3000/queue/{id}
```

Windows (PowerShell):
```powershell
curl.exe http://localhost:3000/queue/{id}
```

Response: `{"info": {"state": "completed", "queue": "quickstart-oneshot_default", "result": {"result": "HELLO WORLD"}}}`

**Step 8: Monitor**

Visit http://localhost:3000/monitoring/

## CLI

runqy includes a CLI for managing queues, tasks, and workers locally or remotely.

```bash
runqy queue list          # List queues
runqy task enqueue -q myqueue -p '{"data":"value"}'
runqy worker list         # List workers
```

See [CLI Reference](https://docs.runqy.com/server/cli/) for full documentation.

## Configuration

Configure via environment variables or YAML files. Key variables:
- `REDIS_HOST`, `REDIS_PASSWORD` - Redis connection
- `RUNQY_API_KEY` - API authentication
- `QUEUE_WORKERS_DIR` - Path to queue YAML configs

See [Configuration Reference](https://docs.runqy.com/server/configuration/) for full documentation.

## See Also

- [runqy-worker](https://github.com/Publikey/runqy-worker) - Task processor ([Docker images](https://ghcr.io/publikey/runqy-worker))
- [runqy-python](https://github.com/Publikey/runqy-python) - Python SDK
- [Documentation](https://docs.runqy.com) - Full documentation

## License

MIT License - see [LICENSE](LICENSE) for details.
