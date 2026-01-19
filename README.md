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

### CLI Alternative

```shell
go install github.com/hibiken/asynq/tools/asynq@latest
asynq dash
```

![Asynq Dashboard](https://github.com/hibiken/asynq/blob/master/docs/assets/dash.gif)

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
