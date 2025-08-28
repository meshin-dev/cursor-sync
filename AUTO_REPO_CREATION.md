# Automatic Repository Creation Feature

## Overview

Cursor-sync now includes automatic repository creation functionality. When you provide a GitHub repository URL that doesn't exist, cursor-sync will automatically create it as a private repository using your GitHub Personal Access Token.

## How It Works

1. **Repository Detection**: When attempting to clone a repository, cursor-sync detects if it doesn't exist
2. **Automatic Creation**: Uses the GitHub API to create the repository as private
3. **Retry Logic**: Implements exponential backoff (max 10 seconds) to handle GitHub's API delay
4. **Security**: Always creates repositories as private to protect sensitive settings

## Features

### 🔒 Security First

- **Always Private**: Repositories are created as private by default
- **Token Authentication**: Uses your GitHub Personal Access Token
- **Secure Description**: Includes security-focused repository description

### 🚀 Smart Retry Logic

- **Exponential Backoff**: Starts with 2-second delays, up to 10 seconds maximum
- **Multiple Attempts**: Up to 5 retry attempts
- **API Delay Handling**: Waits for GitHub to fully initialize the repository

### 📝 Automatic Initialization

- **README.md**: Creates a comprehensive README with security notes
- **Gitignore**: Adds Node.js gitignore template
- **Initial Commit**: Sets up the repository with proper structure

## Usage

### Basic Setup

1. **Set your GitHub token**:

   ```bash
   cursor-sync token ghp_your_token_here
   ```

2. **Configure with any repository URL** (will be created if it doesn't exist):

   ```yaml
   repository:
     url: "https://github.com/yourusername/cursor-settings.git"
   ```

3. **Run sync** - the repository will be created automatically:

   ```bash
   cursor-sync sync
   ```

### Organization Repositories

You can also create repositories in organizations:

```yaml
repository:
  url: "https://github.com/yourorg/cursor-settings.git"
```

**Note**: Your GitHub token must have organization repository creation permissions.

## Error Handling

### Common Scenarios

1. **Repository Already Exists**: Proceeds with normal clone operation
2. **Invalid Token**: Clear error message with authentication guidance
3. **Insufficient Permissions**: Detailed error for organization access issues
4. **Network Issues**: Retry logic handles temporary connectivity problems

### Fallback Behavior

If repository creation fails:

- Clear error messages explain the issue
- No partial state is left behind
- User can manually create the repository and retry

## Configuration

The feature works with existing configuration. No additional settings are required:

```yaml
repository:
  url: "https://github.com/yourusername/cursor-settings.git"
  local_path: "~/.cursor-sync/settings"
  branch: "main"

sync:
  debounce_time: "10s"  # Used for retry delays
  # ... other settings
```

## Testing

Use the provided test script to verify the feature:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./test_auto_repo_creation.sh
```

## Security Considerations

### Repository Privacy

- **Always Private**: Repositories are created as private by default
- **No Public Option**: Cannot accidentally create public repositories
- **Security Description**: README includes security warnings

### Token Permissions

Required GitHub token scopes:

- `repo` - Full control of private repositories
- `org:write` - For organization repositories (if applicable)

### Data Protection

- **Sensitive Settings**: Cursor settings may contain API keys and personal data
- **Private by Design**: Automatic creation ensures privacy
- **Clear Documentation**: README explains security implications

## Troubleshooting

### Repository Creation Fails

1. **Check Token Permissions**:

   ```bash
   cursor-sync token --verify
   ```

2. **Verify Organization Access** (for org repos):
   - Ensure token has organization write permissions
   - Check organization membership

3. **Network Issues**:
   - Retry logic handles most temporary issues
   - Check GitHub API status

### Clone Still Fails After Creation

1. **Wait for GitHub**: Sometimes takes a few seconds for repository to be fully ready
2. **Check Repository**: Verify it was created at the expected URL
3. **Manual Retry**: Run `cursor-sync sync` again

## Implementation Details

### GitHub API Integration

- Uses GitHub REST API v3
- Proper error handling for all HTTP status codes
- Organization vs user repository detection

### Retry Algorithm

- **Base Delay**: 2 seconds
- **Max Delay**: 10 seconds
- **Exponential Backoff**: Delay doubles with each attempt
- **Max Attempts**: 5 total attempts

### Repository Setup

- **Auto-init**: Enables README creation
- **Gitignore**: Node.js template for common exclusions
- **Description**: Security-focused repository description
- **Branch**: Creates main branch with initial commit

## Future Enhancements

Potential improvements:

- Custom repository templates
- Configurable retry parameters
- Support for other Git providers (GitLab, Bitbucket)
- Repository naming conventions
- Automatic branch protection rules
