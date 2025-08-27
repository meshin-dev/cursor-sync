# Cursor Settings Sync

A tool to automatically synchronize Cursor IDE settings between multiple devices using a GitHub Gist.

## Overview

Cursor Settings Sync provides a seamless way to keep your Cursor IDE settings, keybindings, and extensions synchronized across multiple devices. It uses a GitHub Gist as the central storage and automatically detects and syncs changes in both directions.

Unlike VS Code, Cursor currently doesn't have a built-in settings sync feature. This tool bridges that gap by creating symlinks between your local Cursor settings and a GitHub Gist repository, allowing you to maintain consistent settings across all your devices.

## Features

- **Cross-Platform Support**: Works on both macOS and Windows
- **Two-Way Synchronization**: Detects and syncs changes from local to remote and vice versa
- **User-Friendly Notifications**: Prompts before making any changes
- **Conflict Resolution**: Automatically handles merge conflicts
- **Extension Synchronization**: Keeps your installed extensions in sync
- **Automatic Setup**: Creates necessary symlinks and initial configuration
- **Detailed Logging**: Maintains a log of all sync activities
- **Smart Change Detection**: Ignores whitespace-only changes to prevent unnecessary sync prompts
- **Change Preview**: Shows a diff of changes before pushing or pulling

## Prerequisites

- Git installed and configured
- GitHub account
- Cursor IDE installed
- Bash shell (native on macOS, Git Bash or WSL on Windows)

## Quick Installation

The easiest way to install Cursor Settings Sync is using the provided installation script:

1. Create a GitHub Gist as described in the next section
2. Run the installation script with your Gist URL:

```bash
./cursor-sync/install.sh https://gist.github.com/yourusername/your-gist-id
```

The script will:
- Clone your Gist repository
- Set up the necessary symlinks
- Configure automatic startup based on your OS
- Start the sync service

## Manual Installation

### 1. Create a GitHub Gist

1. Go to [https://gist.github.com/](https://gist.github.com/)
2. Create a new **secret** gist with the following files:
   - `settings.json` (can be empty initially)
   - `keybindings.json` (can be empty initially)
3. Note the Gist ID from the URL (e.g., `https://gist.github.com/yourusername/abcd1234efgh5678ijkl`)

### 2. Clone the Gist

```bash
# Create a directory for your Cursor settings
mkdir -p ~/cursor-settings

# Clone the Gist repository
git clone https://gist.github.com/yourusername/your-gist-id.git ~/cursor-settings
```

### 3. Download the Script

1. Clone this repository or download the `cursor-sync` directory
2. Make the script executable:

```bash
chmod +x cursor-sync/cursor-sync.sh
```

### 4. Run the Script

```bash
./cursor-sync/cursor-sync.sh
```

The script will perform an initial setup, creating symlinks between your Cursor settings and the Gist repository.

## Setting Up Automatic Startup

### macOS

1. Copy the provided `com.user.cursorsync.plist` file to your LaunchAgents directory:

```bash
cp cursor-sync/com.user.cursorsync.plist ~/Library/LaunchAgents/
```

2. Edit the file to update the path to your script:

```bash
sed -i '' "s|/path/to/cursor-sync.sh|$(pwd)/cursor-sync/cursor-sync.sh|g" ~/Library/LaunchAgents/com.user.cursorsync.plist
```

3. Load the LaunchAgent:

```bash
launchctl load ~/Library/LaunchAgents/com.user.cursorsync.plist
```

### Windows

1. Edit the provided `cursor-sync-startup.bat` file to update the path to your script
2. Add this batch file to your startup folder:
   - Press `Win+R`
   - Type `shell:startup`
   - Copy your batch file to this folder

## How It Works

1. **Initial Setup**: The script creates symlinks from your Cursor settings files to the cloned Gist repository.
2. **Startup Behavior**: After installation, the script waits for the full sync interval (20 minutes) before performing its first check to avoid immediate prompts.
3. **Periodic Checks**: Every 20 minutes, the script checks for changes in both local and remote settings.
4. **Smart Change Detection**: The script ignores whitespace-only changes, preventing unnecessary sync prompts.
5. **Change Detection**: When significant changes are detected, you'll receive a notification asking if you want to sync.
6. **Change Preview**: Before confirming a sync, you'll see a diff showing exactly what changes will be pushed or pulled.
7. **Synchronization**: The script handles pushing local changes to GitHub or pulling remote changes to your local machine.
8. **Conflict Resolution**: If conflicts occur, the script automatically resolves them by keeping your local version and creating backups.

## File Locations

### macOS
- Settings: `~/Library/Application Support/Cursor/User/settings.json`
- Keybindings: `~/Library/Application Support/Cursor/User/keybindings.json`

### Windows
- Settings: `%APPDATA%\Cursor\User\@settings.json`
- Keybindings: `%APPDATA%\Cursor\User\keybindings.json`

## Customization

You can customize the script by editing the following variables:

- `SYNC_INTERVAL`: Time in seconds between sync checks (default: 300 seconds / 5 minutes)
- `GIST_DIR`: Location of the cloned Gist repository (default: `~/cursor-settings`)

## Troubleshooting

### Logs

Check the log file for detailed information:

```bash
cat ~/cursor-settings/sync.log
```

The log file contains timestamps and detailed information about what the script is doing, including when it's skipping checks, detecting changes, and performing synchronization.

### Common Issues

1. **Authentication Issues**: Ensure you have proper Git credentials set up for pushing to GitHub.
   - For HTTPS URLs, you might need a personal access token
   - For SSH, make sure your SSH key is added to GitHub

2. **Symlink Creation Fails**: On Windows, you may need to run the script as Administrator to create symlinks.

3. **Changes Not Detected**: Make sure the script is running in the background. Check the log file for any errors.
   - On macOS, verify the LaunchAgent is loaded with: `launchctl list | grep cursorsync`
   - On Windows, check if the script is running in Task Manager

4. **Immediate Sync Prompts**: If you're getting sync prompts immediately after installation:
   - Check the log file to see what's happening
   - Verify that the `.last_hash` file was created in your `~/cursor-settings` directory
   - You can manually create or update this file with: `cd ~/cursor-settings && git rev-parse HEAD > .last_hash`

5. **Popup Dialogs Not Responding**: If you dismiss a popup by clicking the X button instead of one of the buttons, the script will continue running normally.

6. **JSON Formatting Differences**: The script now intelligently handles JSON formatting differences and will only prompt for sync when there are actual content changes, not just formatting changes.

## Security Considerations

- The script uses a secret GitHub Gist, which is only accessible to you and people you explicitly share it with.
- Your settings may contain sensitive information like API keys. Always use a private Gist.
- Git credentials are stored according to your Git configuration.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.