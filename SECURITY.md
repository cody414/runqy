# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| Latest  | ✅ Yes             |
| < Latest | ❌ No (upgrade recommended) |

We recommend always running the latest version of Runqy.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them privately:

1. **Email:** [contact@runqy.com](mailto:contact@runqy.com)
2. **GitHub:** Use [private vulnerability reporting](https://github.com/Publikey/runqy/security/advisories/new)

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response timeline

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 7 days
- **Fix or mitigation:** Depends on severity, targeting 30 days for critical issues

### What to expect

- We'll acknowledge your report promptly
- We'll work with you to understand and validate the issue
- We'll credit you in the advisory (unless you prefer anonymity)
- We'll coordinate disclosure timing with you

## Scope

This policy covers:
- [Runqy Server](https://github.com/Publikey/runqy)
- [Runqy Worker](https://github.com/Publikey/runqy-worker)
- [Runqy Python SDK](https://github.com/Publikey/runqy-python)

## Best Practices for Users

- Always use HTTPS for API communication
- Rotate your `RUNQY_API_KEY` regularly
- Use vault for sensitive environment variables
- Keep Runqy and its dependencies up to date
- Restrict network access to the monitoring dashboard
