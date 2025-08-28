# 📦 Repository Setup Guide

This guide explains how to set up your **cursor-sync-bucket** repository for storing your Cursor IDE settings.

## 🎯 Repository Naming Convention

cursor-sync uses a clear naming convention to distinguish between components:

- **`cursor-sync`** = The sync tool itself (this repository)
- **`cursor-sync-bucket`** = Your settings storage repository (what you create)

## 🚀 Quick Setup

### 1. Create Your Bucket Repository

1. Go to [GitHub](https://github.com) and create a **NEW PRIVATE REPOSITORY**
2. Name it: `cursor-sync-bucket`
3. **IMPORTANT**: Set it to **Private** (protect your sensitive settings)
4. Initialize with a README (optional)

### 2. Get Your Repository URL

Your repository URL will be:

```bash
https://github.com/<your-username>/cursor-sync-bucket.git
```

**Example**: If your GitHub username is `johndoe`, your URL is:

```bash
https://github.com/johndoe/cursor-sync-bucket.git
```

## 🔧 Using with cursor-sync

### Interactive Setup (Recommended)

```bash
./bin/cursor-sync setup
# The wizard will prompt for your repository URL
# Enter: https://github.com/<your-username>/cursor-sync-bucket.git
```

### Manual Setup

```bash
# Copy template configuration
cp config/sync.example.yaml config/sync.yaml

# Edit the file and replace:
url: "https://github.com/<your-username>/cursor-sync-bucket.git"
# With your actual username, e.g.:
url: "https://github.com/johndoe/cursor-sync-bucket.git"
```

## 📁 What Gets Stored

Your `cursor-sync-bucket` repository will contain:

- `User/settings.json` - Your Cursor settings
- `User/keybindings.json` - Custom keybindings  
- `User/snippets/` - Code snippets
- `User/tasks.json` - Task configurations
- `User/launch.json` - Debug configurations
- Extensions and other Cursor configuration files

## 🔒 Security Best Practices

1. **Always use a PRIVATE repository** - your settings may contain sensitive information
2. **Never share your repository URL** - keep your settings private
3. **Use a GitHub Personal Access Token** - more secure than username/password
4. **Regularly review repository access** - ensure only you have access

## 🛠 Troubleshooting

### Repository Not Found

```bash
# Check your repository URL format
https://github.com/<username>/cursor-sync-bucket.git
# Ensure the repository exists and is accessible to your GitHub token
```

### Permission Denied

```bash
# Ensure your GitHub token has 'repo' scope
# Verify the repository is private and you own it
```

### Repository is Public

```bash
# cursor-sync will block public repositories for security
# Go to your repository Settings → Change visibility to Private
```

---

**Ready to sync?** Run `cursor-sync setup` to get started! 🚀
