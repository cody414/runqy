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

## Requirements

- Go 1.24+
- Redis
- PostgreSQL (only for production - SQLite is embedded for development)

## Quick Start

**Step 1: Clone the repos**
```bash
git clone https://github.com/Publikey/runqy.git
git clone https://github.com/Publikey/runqy-worker.git
```

**Step 2: Start Redis**
```bash
docker run -d --name redis -p 6379:6379 redis:alpine
```

**Step 3: Start the server**

Linux/Mac:
```bash
cd runqy/app
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=""
export RUNQY_API_KEY=dev-api-key
go run . serve --sqlite
```

Windows (PowerShell):
```powershell
cd runqy/app
$env:REDIS_HOST = "localhost"
$env:REDIS_PORT = "6379"
$env:REDIS_PASSWORD = ""
$env:RUNQY_API_KEY = "dev-api-key"
go run . serve --sqlite
```

**Step 4: Deploy the example queues** (in another terminal)

Linux/Mac:
```bash
cd runqy/app
go build -o runqy .
./runqy login -s http://localhost:3000 -k dev-api-key
./runqy config create -f ../examples/quickstart.yaml
```

Windows (PowerShell):
```powershell
cd runqy/app
go build -o runqy.exe .
.\runqy.exe login -s http://localhost:3000 -k dev-api-key
.\runqy.exe config create -f ..\examples\quickstart.yaml
```

This deploys two example queues:
- `quickstart-oneshot` - spawns a new Python process per task
- `quickstart-longrunning` - keeps Python process alive between tasks

**Step 5: Start a worker** (in another terminal)
```bash
cd runqy-worker
cp config.yml.example config.yml
go run ./cmd/worker
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

Response:
```json
{"info": {"id": "abc123...", "state": "pending", "queue": "quickstart-oneshot_default", ...}, "data": {...}}
```

Use the `id` from the response in the next step.

To try long-running mode, just enqueue to `quickstart-longrunning_default` — the worker already listens on both queues.

**Step 7: Check the result**
```bash
curl http://localhost:3000/queue/{id}/quickstart-oneshot_default
```

Response: `{"info": {"state": "completed", "result": {"result": "HELLO WORLD"}}}`

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

- [runqy-worker](https://github.com/Publikey/runqy-worker) - Task processor
- [runqy-python](https://github.com/Publikey/runqy-python) - Python SDK
- [Documentation](https://docs.runqy.com) - Full documentation

## License

MIT License - see [LICENSE](LICENSE) for details.
