# Rsync-like Sync Functionality

## Overview

Cursor-sync now implements rsync-like logic when copying files from the repository to local Cursor settings. This ensures that only files that have actually changed are copied, preventing unnecessary IDE reactions and improving performance.

## How It Works

### **Current Process (Before)**

1. Pull from remote → Updates `~/.cursor-sync/settings` (repository)
2. Copy ALL files from repository to Cursor settings folder
3. **Problem**: Every file is copied regardless of whether it changed

### **New Process (After)**

1. Pull from remote → Updates `~/.cursor-sync/settings` (repository)
2. **Smart Copy**: Only copy files that have actually changed
3. **Benefit**: Minimal IDE reactions, faster sync, better performance

## Rsync Logic

The system uses a multi-layered approach to determine if a file should be copied:

### **1. File Existence Check (Fastest)**

- If destination file doesn't exist → **Copy**
- If destination file exists → Continue to next check

### **2. File Size Check (Fast)**

- If source and destination have different sizes → **Copy**
- If sizes are identical → Continue to next check

### **3. Modification Time Check (Medium)**

- If source is newer than destination → **Copy**
- If source is older than destination → **Skip**
- If times are equal → Continue to content comparison

### **4. Content Comparison (Slowest but Most Accurate)**

- If file contents are different → **Copy**
- If file contents are identical → **Skip**

## Performance Benefits

### **Reduced IDE Reactions**

- Only changed files trigger IDE reloads
- Unchanged files are left untouched
- Prevents unnecessary IDE restarts and reloads

### **Faster Sync Operations**

- Skips unchanged files entirely
- Reduces I/O operations
- Faster sync completion times

### **Better User Experience**

- Less disruption during sync operations
- IDE remains responsive
- No unnecessary file system events

## Implementation Details

### **File Comparison Algorithm**

```go
func shouldCopyFile(srcPath, destPath string, srcInfo os.FileInfo) bool {
    // 1. Check if destination exists
    destInfo, err := os.Stat(destPath)
    if err != nil {
        return true // Copy if destination doesn't exist
    }

    // 2. Check file size
    if srcInfo.Size() != destInfo.Size() {
        return true // Copy if sizes differ
    }

    // 3. Check modification time
    if srcInfo.ModTime().After(destInfo.ModTime()) {
        return true // Copy if source is newer
    }

    // 4. Content comparison for equal times
    if srcInfo.ModTime().Equal(destInfo.ModTime()) {
        return filesDiffer(srcPath, destPath)
    }

    return false // Skip if source is older
}
```

### **Content Comparison**

```go
func filesDiffer(file1, file2 string) bool {
    data1, err1 := os.ReadFile(file1)
    data2, err2 := os.ReadFile(file2)
    
    if err1 != nil || err2 != nil {
        return true // Assume different if can't read
    }
    
    return !bytes.Equal(data1, data2)
}
```bash

## Usage Examples

### **Scenario 1: New File**

```bash
Source: settings.json (new file)
Destination: settings.json (doesn't exist)
Result: ✅ Copy (file doesn't exist)
```

### **Scenario 2: Modified File**

```bash
Source: settings.json (modified, newer timestamp)
Destination: settings.json (older timestamp)
Result: ✅ Copy (source is newer)
```

### **Scenario 3: Unchanged File**

```bash
Source: keybindings.json (same content, same timestamp)
Destination: keybindings.json (same content, same timestamp)
Result: ⏭️ Skip (no changes detected)
```

### **Scenario 4: Same Size, Different Content**

```bash
Source: snippets.json (100 bytes, content A)
Destination: snippets.json (100 bytes, content B)
Result: ✅ Copy (content differs despite same size)
```

## Logging and Monitoring

### **Sync Statistics**

The system provides detailed logging of sync operations:

```bash
INFO[2025-08-28 01:39:22] 📊 Repository sync completed: 3 files copied, 15 files skipped
DEBU[2025-08-28 01:39:22] 📄 Copied changed file: User/settings.json
DEBU[2025-08-28 01:39:22] ⏭️  Skipped unchanged file: User/keybindings.json
```

### **Debug Information**

Enable debug logging to see detailed file comparison results:

```yaml
logging:
  level: "debug"
```

## Configuration

No additional configuration is required. The rsync-like functionality is enabled by default and works with existing settings:

```yaml
sync:
  debounce_time: "10s"  # Used for sync timing
  # ... other settings

cursor:
  exclude_paths:
    - "*.sock"  # Socket files are still excluded
    # ... other exclusions
```

## Testing

### **Manual Testing**

Create test files and observe sync behavior:

```bash
# Create a test file
echo '{"test": "content"}' > "/Users/developer/Library/Application Support/Cursor/User/test.json"

# Run sync (should copy the new file)
./bin/cursor-sync sync

# Run sync again (should skip unchanged files)
./bin/cursor-sync sync

# Modify the file
echo '{"test": "modified"}' > "/Users/developer/Library/Application Support/Cursor/User/test.json"

# Run sync (should copy only the modified file)
./bin/cursor-sync sync
```

### **Automated Testing**

Use the provided test script:

```bash
./test_rsync_sync.sh
```

## Benefits Summary

### **For Users**

- ✅ **Faster Sync**: Only changed files are processed
- ✅ **Less Disruption**: Minimal IDE reactions
- ✅ **Better Performance**: Reduced I/O operations
- ✅ **Reliable**: Accurate change detection

### **For System**

- ✅ **Efficient**: Optimized file operations
- ✅ **Scalable**: Works with large settings directories
- ✅ **Robust**: Handles edge cases gracefully
- ✅ **Maintainable**: Clear, well-documented logic

## Future Enhancements

Potential improvements:

- **Checksum-based comparison**: Use SHA256 for faster content comparison
- **Incremental sync**: Track file hashes to avoid repeated comparisons
- **Parallel processing**: Copy multiple files simultaneously
- **Compression**: Compress files during transfer
- **Delta sync**: Only transfer file differences

## Troubleshooting

### **Files Not Being Copied**

1. Check file permissions
2. Verify file timestamps
3. Enable debug logging to see comparison results

### **Performance Issues**

1. Ensure exclude paths are properly configured
2. Check for large files that might slow down comparison
3. Monitor system resources during sync

### **IDE Reactions**

1. Verify only changed files are being copied
2. Check IDE file watching settings
3. Review sync logs for unexpected file operations
