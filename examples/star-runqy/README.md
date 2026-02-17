# Star Runqy — Vault Secrets Tutorial

A hands-on example showing how to use Runqy's built-in vault for secret management. The task stars the [Publikey/runqy](https://github.com/Publikey/runqy) repository using a GitHub token stored securely in the vault.

## Prerequisites

- Runqy server running with vault enabled
- A GitHub Personal Access Token with `public_repo` scope

### Enable the Vault

The vault requires a 32-byte AES-256 master key, base64-encoded:

```bash
# Generate a master key
openssl rand -base64 32
```

Set `RUNQY_VAULT_MASTER_KEY` in your server environment (e.g. in `docker-compose.yml`):

```yaml
environment:
  RUNQY_VAULT_MASTER_KEY: "your-base64-key-here"
```

When configured correctly, the server startup banner shows `Vaults: enabled`.

## Step 1: Create a Vault and Store Your Token

```bash
# Create a vault for GitHub credentials
runqy vault create github -d "GitHub API tokens"

# Store your token
runqy vault set github GITHUB_TOKEN ghp_your_token_here

# Verify
runqy vault entries github
```

## Step 2: Deploy the Queue

The queue configuration references the vault by name. All entries in the vault are injected as environment variables into the worker:

```yaml
# examples.yaml
queues:
  star-runqy:
    deployment:
      git_url: "https://github.com/Publikey/runqy.git"
      code_path: "examples/star-runqy"
      startup_cmd: "python main.py"
      mode: "one_shot"
      vaults: ["github"]  # Injects all entries as env vars
```

The worker receives `GITHUB_TOKEN` as an environment variable — no secrets in your code or config files.

Copy `examples.yaml` to your queue workers directory, or use `runqy config create`:

```bash
runqy config create -f examples/star-runqy/examples.yaml
```

## Step 3: Run the Task

```bash
# Enqueue
runqy task enqueue -q star-runqy

# Check result
runqy task list -q star-runqy
```

### Expected Output

```json
{
  "success": true,
  "message": "🌟 Successfully starred Publikey/runqy! Thank you for the support!",
  "emoji_message": "You just starred Runqy from a Runqy task! 🌀"
}
```

## How It Works

1. **`runqy vault set github GITHUB_TOKEN ghp_...`** — encrypts and stores the token (AES-256)
2. **`vaults: ["github"]`** in the queue config — tells the server which vaults to resolve
3. When the worker registers, the server decrypts vault entries and sends them as env vars
4. The Python task reads `os.getenv('GITHUB_TOKEN')` — no vault SDK needed

## Vault CLI Reference

```bash
runqy vault create <name> [-d "description"]   # Create a vault
runqy vault set <name> <key> <value>            # Add/update an entry
runqy vault get <name> <key>                    # Read an entry
runqy vault entries <name>                      # List all entries
runqy vault list                                # List all vaults
runqy vault unset <name> <key>                  # Remove an entry
runqy vault delete <name> --force               # Delete a vault
```

## Private Git Repos

You can also use the vault for git authentication with `git_token`:

```yaml
deployment:
  git_url: "https://github.com/your-org/private-repo.git"
  git_token: "vault://github/GIT_TOKEN"  # vault://vault-name/key
  vaults: ["secrets"]
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `Vaults: disabled` in server banner | Set `RUNQY_VAULT_MASTER_KEY` (must be base64-encoded, 32 bytes) |
| `GITHUB_TOKEN not found` | Check vault entry exists: `runqy vault entries github` |
| `401` from GitHub API | Regenerate PAT, update with `runqy vault set github GITHUB_TOKEN <new>` |
| `vaults not configured` | Server needs restart after setting `RUNQY_VAULT_MASTER_KEY` |
