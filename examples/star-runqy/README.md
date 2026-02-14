# Star Runqy - Vault Secrets Tutorial

This example demonstrates how to use Runqy's built-in vault to securely manage secrets and environment variables in your tasks. We'll create a task that stars the Publikey/runqy repository using a GitHub personal access token stored in the vault.

## 🎯 What You'll Learn

- How to configure secrets in Runqy's vault
- How to reference vault secrets in your YAML deployment
- How to access environment variables in your tasks
- Best practices for handling sensitive data

## 🔐 Step 1: Configure the Vault with Your GitHub Token

First, you'll need a GitHub Personal Access Token (PAT) with `public_repo` permissions:

1. Go to [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Give it a descriptive name like "Runqy Star Example"
4. Select the `public_repo` scope (to star repositories)
5. Click "Generate token" and copy the token

Now store it in Runqy's vault:

```bash
# Add your GitHub token to the vault
runqy vault set GITHUB_TOKEN ghp_your_token_here

# Verify it was stored
runqy vault list
```

## 📄 Step 2: YAML Configuration with env_vars

The magic happens in the YAML deployment configuration. Notice how we reference the vault secret:

```yaml
name: star-runqy
runtime: python
git_url: https://github.com/Publikey/runqy
git_path: examples/star-runqy
env_vars:
  GITHUB_TOKEN: "{{ vault.GITHUB_TOKEN }}"  # 🔐 Vault reference
```

Key points:

- **`env_vars`** section defines environment variables for the worker
- **`{{ vault.GITHUB_TOKEN }}`** references the secret stored in the vault
- The worker receives the actual token value, not the vault reference
- Secrets are injected securely at runtime

## 🚀 Step 3: Deploy and Run the Task

1. **Start your Runqy server** (if not already running):
   ```bash
   runqy server start
   ```

2. **Deploy the worker**:
   ```bash
   # Copy the YAML to your deployment directory
   cp examples/star-runqy.yaml deployment/
   
   # Or add it to examples.yaml and restart
   runqy reload
   ```

3. **Enqueue the task**:
   ```bash
   runqy task enqueue -q star-runqy -t star_runqy
   ```

4. **Watch the magic happen**:
   ```bash
   # Check task status
   runqy task list -q star-runqy
   
   # View results in the monitoring dashboard
   open http://localhost:3000/monitoring/
   ```

## 🌀 Expected Result

When the task completes successfully, you should see:

```json
{
  "success": true,
  "already_starred": false,
  "message": "🌟 Successfully starred Publikey/runqy! Thank you for the support!",
  "emoji_message": "You just starred Runqy from a Runqy task! 🌀",
  "repo_url": "https://github.com/Publikey/runqy"
}
```

**You just starred Runqy from a Runqy task! 🌀**

## 🔒 Vault Security Features

Runqy's vault provides several security benefits:

- **Encrypted storage**: Secrets are encrypted at rest
- **Access control**: Only authorized workers can access specific secrets
- **Audit logging**: Track secret usage and access patterns
- **Environment isolation**: Secrets are injected only when needed
- **No code exposure**: Tokens never appear in your source code

## 🛠️ Troubleshooting

### "GITHUB_TOKEN not found in environment"

This means the vault secret isn't being injected. Check:

1. **Vault secret exists**: `runqy vault list`
2. **YAML syntax**: Ensure `{{ vault.GITHUB_TOKEN }}` is quoted
3. **Deployment reload**: Run `runqy reload` after YAML changes
4. **Worker restart**: Stop and restart the worker if needed

### "Failed to star repository: 401"

The GitHub token is invalid or expired:

1. **Regenerate token**: Create a new PAT on GitHub
2. **Update vault**: `runqy vault set GITHUB_TOKEN new_token`
3. **Check permissions**: Ensure the token has `public_repo` scope

### "Failed to star repository: 403"

Rate limiting or insufficient permissions:

1. **Check rate limits**: GitHub API has rate limits
2. **Verify scope**: Token needs `public_repo` or `repo` scope
3. **Repository access**: Ensure the token can access public repositories

## 🎓 Advanced Vault Usage

This example shows basic vault usage. For production workloads, consider:

```yaml
env_vars:
  # Database credentials
  DB_PASSWORD: "{{ vault.POSTGRES_PASSWORD }}"
  DB_USER: "{{ vault.POSTGRES_USER }}"
  
  # API keys
  OPENAI_API_KEY: "{{ vault.OPENAI_KEY }}"
  STRIPE_SECRET: "{{ vault.STRIPE_SECRET }}"
  
  # AWS credentials
  AWS_ACCESS_KEY_ID: "{{ vault.AWS_ACCESS_KEY }}"
  AWS_SECRET_ACCESS_KEY: "{{ vault.AWS_SECRET_KEY }}"
```

## 📚 Next Steps

- **Explore other examples**: Check out the image classifier and data pipeline examples
- **Production deployment**: Learn about scaling and monitoring in the docs
- **Custom workers**: Build your own GPU-accelerated ML tasks
- **Integration**: Connect Runqy to your existing infrastructure

---

**Pro tip**: The vault supports hierarchical keys like `vault.prod.database.password` for organizing secrets by environment and service.

Ready to build something awesome? The vault has your back! 🔐✨