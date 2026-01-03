# Sway Stack System - System Design Document

## Overview

This document outlines the design for a window stack management system for Sway WM, enabling stack-like navigation between recently focused windows (similar to `mod+Tab` behavior in other window managers).

## Current Implementation

### Components

1. **`sway-focus-tracker.sh`** - Background daemon
   - Subscribes to Sway IPC window events
   - Maintains a stack of focused container IDs in `~/.local/state/sway-focus-stack.txt`
   - Removes duplicates (most recent kept)
   - Handles Sway restarts with retry logic

2. **`sway-stack.sh`** - Toggle script
   - Bound to `$mod+Tab` in Sway config
   - Swaps focus between current and previous window
   - Rotates stack entries on toggle
   - Validates window existence before focusing

3. **State Management**
   - Plain text file with one container ID per line
   - Most recent window at top (line 1)
   - Previous window at line 2
   - Maximum 20 entries to prevent unbounded growth

### Current Architecture

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────────┐
│   Sway WM   │────▶│ sway-focus-tracker  │────▶│ stack.txt       │
│  (IPC)      │◀────│ (daemon)            │     │ (state file)    │
└─────────────┘     └─────────────────────┘     └─────────────────┘
        │                                              │
        │           ┌─────────────────────┐           │
        └──────────▶│   sway-stack.sh     │◀──────────┘
                    │   (toggle script)   │
                    └─────────────────────┘
```

### Issues with Current Implementation

1. **Bash Complexity**: Error handling, temporary file management, and IPC parsing in bash is error-prone
2. **Performance**: Multiple `jq` calls and file operations on each focus change
3. **Reliability**: No proper process supervision; daemon may die without restart
4. **Testing**: Difficult to unit test shell scripts
5. **Maintenance**: Bash scripts are hard to maintain for complex logic

## Go CLI Design

### Goals

1. **Single binary** combining tracker and toggle functionality
2. **Improved reliability** with proper error handling and supervision
3. **Better performance** through efficient IPC handling
4. **Testability** with Go's testing framework
5. **Configuration** via CLI flags and/or config file
6. **Observability** with structured logging and metrics

### Architecture

```
┌─────────────┐     ┌─────────────────────────────────┐
│   Sway WM   │────▶│      sway-stack (Go CLI)        │
│  (IPC)      │◀────│                                  │
└─────────────┘     │  ┌─────────────────────────┐    │
                    │  │     IPC Manager         │    │
                    │  │  - Event subscription   │    │
                    │  │  - JSON parsing         │    │
                    │  └─────────────────────────┘    │
                    │                                  │
                    │  ┌─────────────────────────┐    │
                    │  │     Stack Manager       │    │
                    │  │  - LRU cache (max N)    │    │
                    │  │  - State persistence    │    │
                    │  │  - Window validation    │    │
                    │  └─────────────────────────┘    │
                    │                                  │
                    │  ┌─────────────────────────┐    │
                    │  │     Command Handler     │    │
                    │  │  - Toggle (mod+Tab)     │    │
                    │  │  - Previous window      │    │
                    │  │  - List stack           │    │
                    │  └─────────────────────────┘    │
                    └─────────────────────────────────┘
```

### Components

#### 1. IPC Manager
- Handles connection to Sway IPC socket
- Manages event subscriptions (`window`, `workspace`)
- Parses JSON events efficiently
- Reconnects on Sway restart

#### 2. Stack Manager
- In-memory LRU cache of window IDs (configurable size)
- Persists to disk on changes (optional)
- Validates window existence before operations
- Handles duplicate prevention

#### 3. Command Handler
- `daemon`: Run as background service (systemd/user service)
- `toggle`: Switch to previous window (for keybindings)
- `list`: Show current stack with window metadata
- `clear`: Clear the stack
- `status`: Show daemon status and stack statistics

#### 4. Configuration
- Stack size limit (default: 20)
- State file location
- Log level and output
- IPC socket path auto-detection

### Data Structures

```go
type WindowInfo struct {
    ID      int64  `json:"id"`
    Name    string `json:"name"`
    AppID   string `json:"app_id"`
    Class   string `json:"class"`
    Mark    string `json:"mark,omitempty"`
    Focused bool   `json:"focused"`
}

type Stack struct {
    windows []WindowInfo
    maxSize int
    mu      sync.RWMutex
}

type Config struct {
    StackSize   int    `yaml:"stack_size"`
    DBPath      string `yaml:"db_path"`      // BoltDB database path
    LogLevel    string `yaml:"log_level"`
    LogFile     string `yaml:"log_file"`
    IPC Socket  string `yaml:"ipc_socket"` // Auto-detected if empty
}
```

### Storage with BoltDB

BoltDB provides ACID transactions and concurrent read access in a single file, making it ideal for window state storage without the complexity of SQLite.

#### Simple Schema

```go
// Single bucket for stack, storing window IDs as binary (no JSON)
const bucketStack = "stack"

type BoltStorage struct {
    db *bolt.DB
}

// Store just window IDs, not full WindowInfo
func (b *BoltStorage) PushStack(windowID int64) error {
    return b.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte(bucketStack))
        
        // Read current stack (max 20 IDs)
        var stack []int64
        if data := bucket.Get([]byte("ids")); data != nil {
            // 8 bytes per int64
            for i := 0; i < len(data); i += 8 {
                stack = append(stack, int64(binary.LittleEndian.Uint64(data[i:i+8])))
            }
        }
        
        // Deduplicate and prepend
        stack = removeID(stack, windowID)
        stack = append([]int64{windowID}, stack...)
        if len(stack) > 20 {
            stack = stack[:20]
        }
        
        // Store as binary (more efficient than JSON)
        buf := make([]byte, 0, len(stack)*8)
        for _, id := range stack {
            var idBytes [8]byte
            binary.LittleEndian.PutUint64(idBytes[:], uint64(id))
            buf = append(buf, idBytes[:]...)
        }
        
        return bucket.Put([]byte("ids"), buf)
    })
}

func (b *BoltStorage) GetStack() ([]int64, error) {
    var stack []int64
    err := b.db.View(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte(bucketStack))
        data := bucket.Get([]byte("ids"))
        if data == nil {
            return nil
        }
        
        for i := 0; i < len(data); i += 8 {
            stack = append(stack, int64(binary.LittleEndian.Uint64(data[i:i+8])))
        }
        return nil
    })
    return stack, err
}

// Helper
func removeID(ids []int64, target int64) []int64 {
    result := make([]int64, 0, len(ids))
    for _, id := range ids {
        if id != target {
            result = append(result, id)
        }
    }
    return result
}
```

#### File Location
```
~/.local/state/sway-stack.bolt  # Simple, single-purpose database
```

### API Design

```go
// Core interface
type StackManager interface {
    Push(window WindowInfo)
    Pop() (WindowInfo, bool)
    Peek() (WindowInfo, bool)
    PeekPrevious() (WindowInfo, bool)
    Toggle() (WindowInfo, bool) // Focus previous, rotate stack
    List() []WindowInfo
    Clear()
    Size() int
    Save() error
    Load() error
}

// IPC interface  
type IPCManager interface {
    Connect() error
    Subscribe(events []string) (chan Event, error)
    GetTree() (*Tree, error)
    FocusWindow(id int64) error
    Close() error
}
```

## Implementation Plan

### Phase 1: Core Library
1. **IPC client** for Sway JSON IPC
2. **Stack management** with LRU cache
3. **State persistence** with BoltDB embedded database
4. **Basic CLI** with `daemon` and `toggle` commands

### Phase 2: Enhanced Features
1. **Window validation** before focusing
2. **Structured logging** with rotation
3. **Configuration file** support
4. **Health checks** and self-monitoring

### Phase 3: Integration
1. **Systemd service** files
2. **Sway config** migration guide
3. **Performance benchmarks**
4. **Comprehensive testing**

## Migration from Bash to Go

### Step 1: Coexistence
- Install Go binary alongside bash scripts
- Update Sway config to use Go binary for `mod+Tab`
- Keep bash tracker running during transition

### Step 2: Parallel Operation
- Go binary reads existing stack file format
- Both systems can operate independently
- Compare behavior and fix discrepancies

### Step 3: Full Migration
- Remove bash scripts from startup
- Update documentation
- Clean up old state files

## Configuration Examples

### Sway Config
```bash
# Replace existing binding
bindsym $mod+Tab exec sway-stack toggle

# Optional: Start daemon from sway config
exec_always sway-stack daemon --config ~/.config/sway-stack.yaml
```

### Config File (`~/.config/sway-stack.yaml`)
```yaml
stack_size: 20
db_path: ~/.local/state/sway-state.bolt
log_level: info
log_file: ~/.local/state/sway-stack.log

# Optional window filters
exclude_apps:
  - "org.wezfurlong.wezterm"
  - "Alacritty"
  
include_only:
  - "brave-browser"
  - "code"
```

### Systemd Service (`~/.config/systemd/user/sway-stack.service`)
```ini
[Unit]
Description=Sway Stack Manager
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/sway-stack daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

## Performance Considerations

1. **In-memory cache** with periodic persistence
2. **Batch updates** to reduce disk I/O
3. **Efficient IPC** with connection pooling
4. **Selective subscription** to only necessary events
5. **Background validation** of window existence

## Testing Strategy

1. **Unit tests** for stack operations
2. **Integration tests** with mock IPC
3. **End-to-end tests** in containerized Sway
4. **Performance tests** with high event rates
5. **Failure injection** for reliability testing

## Monitoring and Observability

1. **Structured logs** in JSON format
2. **Prometheus metrics**:
   - `sway_stack_size` (gauge)
   - `sway_stack_operations_total` (counter)
   - `sway_focus_changes_total` (counter)
   - `sway_ipc_errors_total` (counter)
3. **Health endpoint** (HTTP or Unix socket)
4. **Debug commands** for troubleshooting

## Open Questions

1. Should we support multiple stacks (per-workspace, per-output)?
2. How to handle floating windows vs tiled windows?
3. Should we integrate with existing marks system?
4. How to handle window destruction (clean up stack)?
5. Should we support keyboard navigation beyond simple toggle?

## Success Metrics

1. **Reliability**: 99.9% uptime in production use
2. **Performance**: <10ms latency for toggle operations
3. **Memory**: <50MB RSS for daemon
4. **CPU**: <1% average usage
5. **User satisfaction**: Natural feel similar to other WM toggle behavior

## Next Steps

1. [ ] Create Go project structure
2. [ ] Implement basic IPC client
3. [ ] Implement stack manager
4. [ ] Create CLI skeleton
5. [ ] Add configuration management
6. [ ] Write migration tool from bash format
7. [ ] Create test suite
8. [ ] Benchmark against bash implementation
9. [ ] Document installation and usage
10. [ ] Create systemd service files