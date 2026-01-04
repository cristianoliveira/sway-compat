# Sway Stack System - System Design Document

## Overview

This document outlines the design and implementation of a window stack management system for Sway WM, enabling stack-like navigation between recently focused windows (similar to `Alt+Tab` behavior in traditional desktop environments).

## ✅ Current Implementation (IPC-Based Architecture)

### Components

1. **Daemon (`sway-compat daemon`)** - Single unified daemon
   - Subscribes to Sway window focus events via swayipc
   - Maintains in-memory stack of WindowInfo structures
   - Runs Unix domain socket IPC server
   - Handles requests from stack commands (toggle/list/clear)
   - Persists stack to JSON file every 30 seconds
   - Graceful shutdown with signal handling

2. **IPC Server (`pkg/ipc/server.go`)** - Daemon's IPC server
   - Listens on Unix domain socket (`$XDG_RUNTIME_DIR/sway-compat.sock`)
   - Line-delimited JSON protocol
   - Handles concurrent client connections
   - Request types: TOGGLE, LIST, CLEAR, PUSH

3. **Stack Commands** - Thin IPC clients
   - `sway-compat stack toggle` - Switch to previous window
   - `sway-compat stack list` - Show current stack
   - `sway-compat stack clear` - Clear the stack
   - All communicate with daemon via IPC client (`pkg/ipc/daemon_client.go`)

4. **Storage (`pkg/storage/storage.go`)** - JSON file persistence
   - File path: `~/.local/state/sway-compat-stack.json`
   - Simple JSON array of window IDs
   - Unix file locking (flock) for safety
   - Daemon owns storage, periodic saves (not per-operation)

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                  sway-compat daemon                          │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌───────────────┐ │
│  │   Sway IPC   │───▶│    Stack     │───▶│   Storage     │ │
│  │   Client     │    │   Manager    │    │ (JSON file)   │ │
│  │  (swayipc)   │    │ (in-memory)  │    │   + flock     │ │
│  └──────────────┘    └──────────────┘    └───────────────┘ │
│                              │                              │
│                              │                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              IPC Server (Unix socket)                  │ │
│  │           /run/user/1000/sway-compat.sock              │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                             ▲
                             │ IPC requests
                             │ (JSON over socket)
                             │
                ┌────────────┴────────────┐
                │                         │
       ┌────────┴──────┐       ┌─────────┴────────┐
       │ stack toggle  │       │   stack list     │
       │ (IPC client)  │       │  (IPC client)    │
       └───────────────┘       └──────────────────┘
```

## Architecture Benefits

1. **Single Source of Truth**: Daemon owns all state, no race conditions
2. **Performance**: In-memory operations, no file I/O on every command
3. **Reliability**: IPC ensures commands work only if daemon is running
4. **Clean Separation**: Commands are thin clients, logic in daemon
5. **Testability**: Well-defined interfaces, easy to mock
6. **Maintainability**: Go code with proper error handling

## Design Decisions

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

### Storage Implementation

Simple JSON file storage with Unix file locking for concurrent access safety.

#### JSON File Storage

```go
type FileStorage struct {
    path string
    log  logger.Logger
}

// PushStack adds window ID to the stack with file locking
func (f *FileStorage) PushStack(windowID int64) error {
    // Lock file for writing (exclusive lock)
    file, err := f.lockFile(true)
    if err != nil {
        return err
    }
    defer f.unlockFile(file)

    // Read current stack
    stack := f.readStackFromFile(file)

    // Remove duplicates
    stack = removeID(stack, windowID)

    // Prepend new ID
    stack = append([]int64{windowID}, stack...)

    // Trim to max size
    if len(stack) > maxStackSize {
        stack = stack[:maxStackSize]
    }

    // Write back
    return f.writeStackToFile(file, stack)
}

// File locking using Unix flock
func (f *FileStorage) lockFile(write bool) (*os.File, error) {
    flags := os.O_RDONLY
    if write {
        flags = os.O_RDWR | os.O_CREATE
    }

    file, err := os.OpenFile(f.path, flags, 0644)
    if err != nil {
        return nil, err
    }

    // Apply file lock
    lockType := syscall.LOCK_SH // Shared lock for reading
    if write {
        lockType = syscall.LOCK_EX // Exclusive lock for writing
    }

    if err := syscall.Flock(int(file.Fd()), lockType); err != nil {
        file.Close()
        return nil, err
    }

    return file, nil
}
```

#### File Location
```
~/.local/state/sway-compat-stack.json  # Human-readable JSON array
```

**Advantages:**
- Human-readable (can inspect with `cat`)
- Simple implementation
- Unix flock provides process-safe locking
- No database overhead

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

## Implementation Status

### ✅ Phase 1: Core Library (Completed)
- ✅ IPC client for Sway JSON IPC (using swayipc library)
- ✅ Stack management with in-memory cache
- ✅ State persistence with JSON file storage
- ✅ Basic CLI with `daemon` and `stack` commands
- ✅ IPC server for daemon-command communication
- ✅ IPC client for stack commands

### ✅ Phase 2: Enhanced Features (Completed)
- ✅ Window validation before focusing
- ✅ Structured logging with configurable levels
- ✅ Graceful shutdown with signal handling
- ✅ Periodic persistence (every 30 seconds)
- ✅ Exclude/include app filters

### 🔄 Phase 3: Future Enhancements
- [ ] Configuration file support (YAML/TOML)
- [ ] Systemd service files
- [ ] Per-workspace stack support
- [ ] Advanced filtering rules

## Configuration Examples

### Sway Config
```bash
# Window focus stack (Alt+Tab style)
bindsym $mod+Tab exec sway-compat stack toggle

# Start daemon on Sway launch
exec_always sway-compat daemon

# Optional: With custom settings
# exec_always sway-compat daemon --stack-size 10 --exclude waybar,swaylock

# Optional: Enable debug logging
# exec_always env SWAY_COMPAT_LOGS_LEVEL=DEBUG sway-compat daemon
```

### Command-Line Flags

**Daemon:**
```bash
sway-compat daemon [flags]

Flags:
  --db-path string         Path to stack file (default: ~/.local/state/sway-compat-stack.json)
  --stack-size int         Maximum windows to track (default: 20)
  --exclude strings        App IDs to exclude (comma-separated)
  --include-only strings   Only track these app IDs (comma-separated)
```

**Stack commands:**
```bash
sway-compat stack toggle    # Switch to previous window
sway-compat stack list      # Show current stack
sway-compat stack clear     # Clear the stack
```

### Systemd Service (Optional)

`~/.config/systemd/user/sway-compat-daemon.service`
```ini
[Unit]
Description=Sway Compat Daemon (Window Stack Manager)
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/sway-compat daemon
Restart=on-failure
RestartSec=5
Environment="SWAY_COMPAT_LOGS_LEVEL=INFO"

[Install]
WantedBy=default.target
```

Enable with:
```bash
systemctl --user enable sway-compat-daemon.service
systemctl --user start sway-compat-daemon.service
```

**Note:** Most users prefer `exec_always` in Sway config over systemd services.

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

## Implementation Checklist

### ✅ Completed
- [x] Create Go project structure
- [x] Implement Sway IPC client (using swayipc library)
- [x] Implement stack manager with in-memory cache
- [x] Create CLI with Cobra framework
- [x] JSON file storage with Unix file locking
- [x] Create comprehensive test suite
- [x] Document installation and usage
- [x] IPC server for daemon
- [x] IPC client for stack commands
- [x] Window validation before focus
- [x] Graceful shutdown handling
- [x] Exclude/include app filters
- [x] Periodic persistence strategy

### 🔄 Future Work
- [ ] Configuration file support (YAML/TOML)
- [ ] Systemd service templates
- [ ] Per-workspace stack support
- [ ] Advanced filtering and rules engine
- [ ] Performance metrics and monitoring
- [ ] Integration tests with containerized Sway