# Security Policy

## Private Repository Requirement

**🔒 cursor-sync enforces the use of PRIVATE repositories for security.**

### Why Private Repositories Are Required

Cursor IDE settings often contain sensitive information that should never be exposed publicly:

- **API Keys & Tokens**: Many extensions store authentication tokens
- **Personal Workspace Paths**: Local file system paths that reveal your directory structure
- **Custom Snippets**: May contain proprietary code or sensitive templates
- **Extension Configurations**: Could include private server URLs or credentials
- **User Preferences**: Personal settings that may reveal usage patterns

### Security Checks

cursor-sync automatically performs privacy checks:

1. **Installation Time**: Verifies repository privacy before installation
2. **Sync Operations**: Checks privacy before each push operation
3. **Daemon Startup**: Validates repository privacy when daemon initializes

### What Happens with Public Repositories

If a public repository is detected, cursor-sync will:

1. **Display a prominent warning** with security implications
2. **Block all sync operations** to prevent data exposure
3. **Log the security violation** for audit purposes
4. **Provide clear instructions** on how to fix the issue

### Example Warning Message

```
================================================================================
⚠️  SECURITY WARNING: PUBLIC REPOSITORY DETECTED!
================================================================================

Repository: https://github.com/user/public-repo.git

❌ CURSOR SYNC BLOCKED - This repository appears to be PUBLIC!

Why this matters:
• Cursor settings may contain sensitive information (API keys, tokens)
• Personal configurations and extensions could be exposed
• Workspace paths and project details might be leaked

🔒 SOLUTION: Use a PRIVATE repository for syncing Cursor settings

To fix this:
1. Create a new PRIVATE GitHub repository
2. Update config/sync.yaml with the private repository URL
3. Ensure the repository is set to private in GitHub settings
================================================================================
```

### Supported Git Platforms

- **GitHub**: Full privacy verification via public API
- **GitLab**: Privacy checking supported
- **Other Git Services**: Conservative approach - assumes private if verification fails

### Repository Creation Best Practices

1. **Always create repositories as PRIVATE**
2. **Use specific repository names** (e.g., `cursor-settings`, not `settings`)
3. **Limit repository access** to only your accounts
4. **Regular audit** of repository visibility settings
5. **Use SSH keys** for authentication when possible

### Reporting Security Issues

If you discover a security vulnerability in cursor-sync, please report it responsibly:

1. **Do not** open a public issue
2. Contact the maintainers directly
3. Provide detailed information about the vulnerability
4. Allow time for the issue to be addressed before public disclosure

### Security-First Development

cursor-sync is built with security as a primary concern:

- **Fail-safe defaults**: Block operations rather than risk exposure
- **Multiple check points**: Privacy verified at multiple stages
- **Clear error messages**: Help users understand security implications
- **Audit logging**: All security checks are logged for review

## Compliance

This security policy helps ensure:

- Protection of personal and sensitive data
- Compliance with data privacy regulations
- Safe sharing of development environments
- Reduced risk of accidental data exposure

Remember: **Your Cursor settings are personal and potentially sensitive. Keep them private!**
