# Contributing to Runqy

Thanks for your interest in contributing to Runqy! Whether it's a bug report, feature request, documentation improvement, or code contribution — we appreciate it.

## Getting Started

1. **Fork** the repository
2. **Clone** your fork locally
3. **Create a branch** from `main` for your changes
4. **Make your changes** and commit with clear messages
5. **Push** to your fork and open a **Pull Request**

## Development Setup

### Server (Go)

```bash
git clone https://github.com/Publikey/runqy.git
cd runqy
go build -o runqy ./app
```

Requirements:
- Go 1.21+
- Redis (for queue backend)
- PostgreSQL or SQLite (for configuration storage)

### Worker

See [runqy-worker](https://github.com/Publikey/runqy-worker) for worker development.

### Python SDK

See [runqy-python](https://github.com/Publikey/runqy-python) for SDK development.

## Pull Requests

- **One PR per feature/fix** — keep changes focused
- **Write clear commit messages** — explain *why*, not just *what*
- **Add tests** for new functionality when possible
- **Update documentation** if your change affects user-facing behavior
- **Reference related issues** in your PR description (e.g., `Fixes #123`)

### Branch Naming

- `feature/short-description` — new features
- `fix/short-description` — bug fixes
- `docs/short-description` — documentation changes
- `examples/short-description` — new examples

## Bug Reports

Open an [issue](https://github.com/Publikey/runqy/issues/new) with:
- **What happened** vs **what you expected**
- **Steps to reproduce**
- **Environment** (OS, Go version, Redis version, Runqy version)
- **Logs** if relevant

## Feature Requests

Open an [issue](https://github.com/Publikey/runqy/issues/new) describing:
- **The problem** you're trying to solve
- **Your proposed solution**
- **Alternatives** you've considered

## Code Style

- **Go**: Follow standard Go conventions (`gofmt`, `go vet`)
- **Python SDK**: Follow PEP 8
- Keep code readable — clarity over cleverness

## Community

- Be respectful and constructive
- We're building this together — every contribution matters

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
