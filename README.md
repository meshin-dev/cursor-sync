# 🚀 Cursor Sync

**Effortlessly sync your Cursor IDE settings across multiple macOS machines with real-time monitoring, smart debouncing, and intelligent file comparison.**

Keep your Cursor IDE configuration, keybindings, snippets, and all User settings perfectly synchronized across all your development machines automatically.

---

## ✨ What is Cursor Sync?

Cursor Sync automatically keeps your Cursor IDE settings synchronized across multiple machines by:

1. **👀 Watching** your Cursor User configuration files in real-time with smart debouncing
2. **🔄 Syncing** changes to your private Git repository instantly
3. **📥 Pulling** updates from other machines automatically
4. **🧠 Resolving** conflicts intelligently based on content hashes
5. **🔒 Protecting** your sensitive data with private-only repositories
6. **🗑️ Handling** file deletions in both directions automatically

**Perfect for developers who work on multiple machines and want consistent Cursor IDE experience everywhere.**

---

## 🎯 Key Features

### 🚀 **Smart Sync System**

- **⚡ Dual Architecture**: Real-time fsnotify (primary) + periodic intervals (fallback)
- **🧠 Smart Debouncing**: Configurable 10s+ debounce prevents sync storms
- **🤖 Automatic Everything**: Set once, sync everywhere automatically
- **🔄 Hash-Based Comparison**: SHA256 content hashing prevents unnecessary syncs
- **🚀 Auto Repository Creation**: Automatically creates private repositories if they don't exist
- **📊 Rsync-like Sync**: Only copies changed files to prevent unnecessary IDE reactions
- **🗑️ Deletion Sync**: Automatically syncs file deletions in both directions

### 🔒 **Security & Privacy**

- **🛡️ Private Repository Only**: Automatically blocks public repos
- **🔑 GitHub Token Auth**: Secure token-based authentication
- **🔍 Privacy Validation**: Real-time repository privacy checking
- **📋 Cursor Detection**: Validates Cursor installation before setup
- **🔒 Auto-Created Repos**: All automatically created repositories are private by default

### 🎛️ **User Experience**

- **🌟 One-Command Setup**: `cursor-sync bootstrap` does everything
- **📊 Interactive Wizards**: Guided setup with smart defaults
- **📱 Rich CLI**: Intuitive commands with helpful output
- **📝 Comprehensive Logging**: Detailed logs with daily rotation

### 📁 **Complete Coverage (User Folder Only)**

- ✅ **Settings** (`settings.json`) - All your preferences
- ✅ **Keybindings** (`keybindings.json`) - Custom shortcuts
- ✅ **Snippets** (`snippets/`) - Code templates
- ✅ **Tasks & Launch** (`tasks.json`, `launch.json`) - Debug configs
- ✅ **Extensions** (`extensions.json`) - Your installed extensions
- ✅ **Workspace Settings** - Project-specific configurations
- ✅ **All User Files** - Everything in the User folder except excluded paths

**Note**: Only the `/User` folder is synced. Other Cursor folders are excluded to prevent conflicts and infinite loops.

---

## 📋 Prerequisites

### ✅ **Required**

- **macOS** (currently supported platform)
- **Cursor IDE** installed and launched at least once
- **Git** installed on your system
- **GitHub account** with a private repository
- **GitHub Personal Access Token** (the setup wizard helps create this)

### 🔍 **Quick Check**

```bash
cursor-sync check
```

---

## 🚀 **ONE-COMMAND SETUP**

### 🌟 **Bootstrap (Recommended)**

Just run this **one command** and follow the prompts:

```bash
cursor-sync bootstrap
```

**That's it!** The bootstrap wizard will:

1. 🔍 Validate your Cursor IDE installation
2. 🔑 Help you set up your GitHub Personal Access Token  
3. 📦 Configure your private repository (`cursor-sync-bucket`)
4. ⚙️ Create all necessary configuration files
5. 🔧 Install the background daemon
6. 🚀 Start the sync service
7. ✅ Verify everything is working

**No multiple commands, no confusion - just one command that does everything!**

---

## 📖 Manual Setup (Alternative)

If you prefer manual control:

### **Step 1: Get Cursor Sync**

```bash
git clone <this-repository-url>
cd cursor-sync
go build -o bin/cursor-sync .
```

### **Step 2: Create Your Settings Repository**

### Option A: Automatic Creation (Recommended)

- Just provide any repository URL in the setup - cursor-sync will create it automatically as private
- No manual repository creation needed!

### Option B: Manual Creation

1. Go to [GitHub](https://github.com) → Create new **private** repository
2. Name it: `cursor-sync-bucket` (recommended)
3. ⚠️ **Must be private** to protect your settings

### **Step 3: Get GitHub Token**

1. Go to [GitHub Settings](https://github.com/settings/tokens)
2. Generate new token (classic) with **`repo`** scope
3. Copy the token (starts with `ghp_`)

### **Step 4: Interactive Setup**

```bash
./bin/cursor-sync setup
# Follow the prompts to configure token and repository
```

### **Step 5: Install & Start**

```bash
./bin/cursor-sync install
./bin/cursor-sync start
```

---

## 🎛️ Daily Commands

### **Essential Commands**

```bash
# Check everything is working
cursor-sync status

# Manual sync (if needed)  
cursor-sync sync

# View logs
cursor-sync logs

# Pause/resume syncing
cursor-sync pause
cursor-sync resume
```

### **Management Commands**

```bash
# Validate configuration
cursor-sync validate

# Check Cursor installation
cursor-sync check

# Control daemon
cursor-sync start
cursor-sync stop
cursor-sync restart
```

---

## 🚀 Automatic Repository Creation

### **Zero-Setup Repositories**

Cursor-sync can automatically create your settings repository if it doesn't exist:

```bash
# Just provide any repository URL - it will be created automatically!
cursor-sync setup
# Enter: https://github.com/yourusername/cursor-settings.git
# ✅ Repository created automatically as private
```

### **How It Works**

1. **🔍 Detection**: Cursor-sync detects when a repository doesn't exist
2. **🚀 Creation**: Uses GitHub API to create the repository as private
3. **⏳ Retry Logic**: Implements smart retry with exponential backoff (max 10s)
4. **🔒 Security**: Always creates private repositories to protect your data

### **Features**

- **🔒 Always Private**: No risk of accidentally creating public repositories
- **🧠 Smart Retries**: Handles GitHub API delays with exponential backoff
- **📝 Auto-Initialization**: Creates README and proper repository structure
- **🏢 Organization Support**: Works with user and organization repositories

### **Requirements**

- GitHub Personal Access Token with `repo` scope
- For organization repos: `org:write` scope
- Valid repository URL format

---

## 📊 Hash-Based Smart Sync

### **Intelligent File Comparison**

Cursor-sync uses SHA256 content hashing to only sync files that have actually changed:

```bash
# Only changed files are copied - no unnecessary IDE reactions!
cursor-sync sync
# Output: 📊 Repository sync completed: 2 files copied, 15 files skipped
```

### **How It Works**

1. **Content Hashing**: SHA256 hashes of file contents are compared
2. **Hash Caching**: Calculated hashes are cached for performance
3. **Throttled Calculation**: Hash calculations are throttled to prevent CPU stress
4. **Polling Mechanism**: Waits for hash calculations to complete with timeout
5. **Smart Skipping**: Identical files are skipped entirely

### **Benefits**

- ✅ **Faster Sync**: Only changed files are processed
- ✅ **Less Disruption**: Minimal IDE reactions and reloads
- ✅ **Better Performance**: Reduced I/O operations and CPU usage
- ✅ **Accurate**: Content-based change detection eliminates false positives
- ✅ **Throttled**: Prevents CPU stress during bulk operations

---

## 🗑️ File Deletion Sync

### **Bidirectional Deletion Handling**

Cursor-sync automatically handles file deletions in both directions:

```bash
# Local deletion → Remote deletion
rm ~/Library/Application\ Support/Cursor/User/some-file.json
# ✅ File automatically removed from repository

# Remote deletion → Local deletion  
git rm repository/User/some-file.json && git push
# ✅ File automatically removed from local Cursor settings
```

### **How It Works**

1. **Real-time Detection**: File watcher detects local file deletions
2. **Periodic Detection**: Remote deletions detected during periodic syncs
3. **Automatic Cleanup**: Deleted files are removed from target locations
4. **Logging**: All deletion operations are logged with file counts

### **Features**

- ✅ **Real-time Local Deletions**: Immediate sync of local file deletions
- ✅ **Periodic Remote Deletions**: Remote deletions synced during periodic intervals
- ✅ **Safe Operations**: Only synced files are considered for deletion
- ✅ **Detailed Logging**: Clear logs show deletion operations

---

## ⚙️ Configuration

### **Automatic Configuration**

The bootstrap command creates optimal defaults. Your config lives at:

```bash
~/.cursor-sync/config.yaml
```

### **Customization Options**

```yaml
sync:
  pull_interval: "5m"              # How often to check for remote changes
  push_interval: "5m"              # How often to push local changes  
  debounce_time: "10s"             # Minimum 10s debounce for real-time sync
  watch_enabled: true              # Enable real-time file watching
  conflict_resolve: "newer"        # newer|local|remote
  hash_throttle_delay: "100ms"     # Delay between hash calculations
  hash_polling_timeout: "10s"      # Max time to wait for hash calculation

cursor:
  config_path: "~/Library/Application Support/Cursor"
  exclude_paths:
    - "User/globalStorage/"        # Cursor's internal data (causes infinite loops)
    - "logs/"
    - "CachedExtensions/"
    - "**/node_modules/"
    # ... performance-optimized exclusions
```

---

## 🔄 How It Works

### **Sync Behavior**

1. **🔄 Startup Sync**: Always syncs on daemon start/restart
2. **⚡ Real-time Sync**: Detects changes within 10+ seconds (configurable)
3. **🕒 Periodic Backup**: Regular intervals ensure nothing is missed
4. **🆕 Fresh Install Logic**: Overwrites local settings if never synced before
5. **🧠 Smart Conflicts**: Prefers newer commits automatically
6. **🗑️ Deletion Sync**: Handles file deletions in both directions

### **The `.custom.sync` Marker**

- **Purpose**: Indicates if local settings have been synced before
- **Location**: `~/Library/Application Support/Cursor/.custom.sync`
- **Behavior**: Missing marker = fresh install (overwrite local files)

### **Security Features**

- **🛡️ Private Repository Enforcement**: Blocks public repositories automatically
- **🔑 Token-based Auth**: No passwords or SSH keys needed
- **🔍 Real-time Privacy Checks**: Validates repository privacy before every sync
- **📋 Local Validation**: Ensures Cursor is installed and accessible

### **User Folder Focus**

- **Scope**: Only the `/User` folder within Cursor settings is synced
- **Rationale**: Prevents conflicts with Cursor's internal data and other folders
- **Exclusions**: `globalStorage` and other system folders are excluded
- **Benefits**: Clean, focused sync without infinite loops or conflicts

---

## 🔧 Troubleshooting

### **Common Issues**

#### **Sync not working**

```bash
cursor-sync status      # Check daemon status
cursor-sync logs        # View detailed logs  
cursor-sync validate    # Verify configuration
```

#### **Permission issues**

```bash
cursor-sync token show  # Check token status
# Ensure token has 'repo' scope
```

#### **Settings not syncing**

```bash
cursor-sync check       # Verify Cursor installation
ls ~/Library/Application\ Support/Cursor/.custom.sync  # Check sync marker
```

#### **Infinite sync loops**

```bash
# Check if globalStorage is excluded
cat ~/.cursor-sync/config.yaml | grep globalStorage

# Restart daemon if needed
cursor-sync restart
```

### **Reset Everything**

```bash
cursor-sync stop
rm -rf ~/.cursor-sync
cursor-sync bootstrap   # Start fresh
```

---

## 📊 Advanced Features

### **Repository Naming Convention**

- **Tool**: `cursor-sync` (this repository)
- **Storage**: `cursor-sync-bucket` (your settings repository)

Clear separation makes organization simple!

### **Smart Debouncing**

Prevents excessive syncs during rapid changes:

- **Minimum**: 10 seconds (enforced)
- **Default**: 10 seconds (good for most users)  
- **Configurable**: Up to minutes for heavy development

### **Hash Calculation Throttling**

Prevents CPU stress during bulk operations:

- **Throttle Delay**: 100ms between hash calculations (configurable)
- **Polling Timeout**: 10s maximum wait for hash completion
- **Caching**: Hash results are cached for performance

### **Comprehensive Logging**

```bash
cursor-sync logs           # Today's activity
cursor-sync logs --tail    # Real-time monitoring
cursor-sync logs --date 2024-01-15  # Specific date
```

### **Multiple Machine Setup**

Run `cursor-sync bootstrap` on each machine with the same GitHub token and repository. Settings sync automatically!

---

## 🆘 Getting Help

1. **📊 Check Status**: `cursor-sync status`
2. **📋 View Logs**: `cursor-sync logs --tail`
3. **✅ Validate Setup**: `cursor-sync validate`
4. **🔍 Debug Mode**: `cursor-sync --verbose <command>`

---

## 🎉 Success! What Next?

After running `cursor-sync bootstrap`:

1. **✅ Make changes in Cursor** → They sync within 10 seconds
2. **🗑️ Delete files** → Deletions sync automatically
3. **🖥️ Work on another machine** → Run bootstrap there too  
4. **📊 Monitor activity** → `cursor-sync status` and `cursor-sync logs`
5. **⏸️ Pause when needed** → `cursor-sync pause` during major changes

**Your Cursor IDE settings are now protected and synchronized across all your machines!** 🎊

---

## Configuration Files & Management

### 📁 **File Locations**

cursor-sync stores its configuration in several locations. Here's where everything is located:

#### **Main Configuration**

```bash
~/.cursor-sync/config.yaml          # Main configuration file
~/.cursor-sync/.github              # GitHub Personal Access Token (secure)
```

#### **Project Files** (in cursor-sync directory)

```bash
./config/sync.example.yaml          # Template configuration file
./bin/cursor-sync                   # Built binary (ignored by git)
./logs/                            # Application logs (ignored by git)
├── 2024-01-15/                    # Daily log folders
│   ├── cursor-sync.log            # Main application logs
│   └── cursor-sync.error.log      # Error logs only
```

#### **System Integration** (macOS)

```bash
~/Library/LaunchAgents/com.cursor-sync.plist    # macOS daemon configuration
```

#### **Repository Storage** (your private repo)

```bash
<your-repo>/User/                  # Your cursor-sync-bucket repository (User folder only)
├── settings.json                  # Cursor IDE settings
├── keybindings.json              # Keyboard shortcuts
├── snippets/                     # Code snippets
├── tasks.json                    # VS Code tasks
├── launch.json                   # Debug configurations
└── extensions.json               # Extension recommendations
```

### ⚙️ **Managing Configuration**

#### **View Current Configuration**

```bash
# Show all configuration details
cursor-sync validate

# Show only config file validation
cursor-sync config-validate

# Check GitHub token status
cursor-sync token show
```

#### **Edit Configuration**

```bash
# Edit main configuration file
nano ~/.cursor-sync/config.yaml

# View example configuration
cat ./config/sync.example.yaml
```

#### **Change Settings**

```bash
# Update GitHub token
cursor-sync token <new-token>

# Restart daemon to apply changes
cursor-sync stop
cursor-sync start
```

#### **View Logs**

```bash
# Tail live logs
cursor-sync logs tail

# View recent logs  
cursor-sync logs view

# Open logs directory
cursor-sync logs open
```

### 🗑️ **Complete Uninstall**

To completely remove cursor-sync from your system:

```bash
# 1. Stop the daemon
cursor-sync stop

# 2. Remove system integration
rm ~/Library/LaunchAgents/com.cursor-sync.plist

# 3. Remove configuration directory
rm -rf ~/.cursor-sync

# 4. Remove project directory (optional)
cd .. && rm -rf cursor-sync

# 5. Your sync repository remains untouched (your data is safe)
```

**Note**: Your cursor-sync-bucket repository and Cursor settings are NOT deleted during uninstall - they remain safe in your GitHub repository.

### 📝 **Configuration File Reference**

The main configuration file (`~/.cursor-sync/config.yaml`) contains:

```yaml
repository:
  url: "https://github.com/username/cursor-sync-bucket.git"
  branch: "main"
  local_path: "~/.cursor-sync/settings"

cursor:
  config_path: "~/Library/Application Support/Cursor"
  exclude_paths:
    - "User/globalStorage/"        # Cursor's internal data
    - "logs/"
    - "CachedExtensions/"
    - "**/node_modules/"
    # ... other exclusions

sync:
  pull_interval: "5m"              # How often to pull from remote
  push_interval: "5m"              # How often to push local changes
  debounce_time: "10s"             # Minimum time between real-time syncs
  watch_enabled: true              # Enable real-time file watching
  conflict_resolve: "newer"        # Conflict resolution strategy
  hash_throttle_delay: "100ms"     # Delay between hash calculations
  hash_polling_timeout: "10s"      # Max time to wait for hash calculation

logging:
  level: "info"                    # Log level: debug, info, warn, error
  log_dir: "~/.cursor-sync/logs"   # Log directory
  max_size: 10                     # Max size per log file (MB)
  max_days: 30                     # Days to keep logs
  compress: true                   # Compress old logs
```

---

## Troubleshooting

### Common Issues

### "cursor-sync: command not found"

```bash
# Make sure you built the binary
go build -o bin/cursor-sync .

# Or use full path
./bin/cursor-sync --help
```

### "Cursor installation not found"

- Make sure Cursor IDE is installed and has been run at least once
- Check that settings directory exists: `~/Library/Application Support/Cursor/User/`

### "Repository is not private"

- Your sync repository MUST be private for security
- Go to repository Settings → Change visibility → Make private

### "GitHub token invalid"

```bash
# Set a new token
cursor-sync token your-new-token

# Check token status
cursor-sync token show
```

### "Permission denied" errors

```bash
# Check daemon status
cursor-sync status

# Restart daemon
cursor-sync stop
cursor-sync start
```

### "Sync conflicts"

- cursor-sync automatically resolves conflicts by timestamp (newer wins)
- Check logs for details: `cursor-sync logs tail`

### "Infinite sync loops"

```bash
# Check if globalStorage is excluded
cat ~/.cursor-sync/config.yaml | grep globalStorage

# Restart daemon
cursor-sync restart
```

### "Configuration issues"

```bash
# Validate configuration
cursor-sync validate

# Reset to defaults (backup your settings first!)
rm ~/.cursor-sync/config.yaml
cursor-sync setup  # Reconfigure
```

---

## License

MIT License - see LICENSE file for details.

---

*Built with ❤️ for developers who value consistency and automation.*
