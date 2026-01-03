# Sway Window Cycling System - System Design Document

## Overview

This document outlines the design for a window cycling system for Sway WM that emulates macOS's `Mod+` behavior - cycling through windows of the same application. The system allows users to quickly switch between multiple windows of the same application (e.g., cycling through 3 Firefox windows) without leaving the current workspace context.

## Current Implementation

### Components

1. **`sway-cycle-app.sh`** - Bash script
   - Queries Sway IPC for window tree
   - Identifies currently focused window's application via `app_id` or `window_properties.class`
   - Finds all visible windows with the same identifier across workspaces
   - Cycles to next window in circular fashion
   - Excludes scratchpad/hidden windows
   - Includes debug logging capabilities

### Current Architecture

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────────┐
│   Sway WM   │────▶│ sway-cycle-app.sh   │────▶│   Window ID     │
│  (IPC)      │◀────│ (bash script)       │     │   Matching      │
└─────────────┘     └─────────────────────┘     └─────────────────┘
        │                                                  │
        │           ┌─────────────────────┐               │
        └──────────▶│  Focus Next Window  │◀──────────────┘
                    │     (cycling)       │
                    └─────────────────────┘
```

### Current Logic Flow

1. **Get focused window**: Query Sway IPC for currently focused window
2. **Extract identifier**: Get `app_id` or `window_properties.class`
3. **Find matching windows**: Locate all visible windows with same identifier
4. **Calculate next window**: Find current position in list, move to next (wrap around)
5. **Focus window**: Send focus command via Sway IPC

### Issues with Current Implementation

1. **Performance**: Full tree query and JSON parsing on each keypress
2. **Error handling**: Basic error handling in bash
3. **No state management**: Always queries fresh tree, no caching
4. **Testing difficulty**: Hard to unit test bash scripts
5. **Configuration**: Hardcoded behavior, no runtime configuration
6. **Limited features**: Only forward cycling, no backward cycling option

## Go CLI Design

### Goals

1. **Single binary** with multiple operating modes
2. **Improved performance** through caching and efficient IPC
3. **Bidirectional cycling** (forward/backward)
4. **Configurable behavior** via CLI flags and config file
5. **Enhanced window matching** with regex patterns and filters
6. **Observability** with structured logging and metrics
7. **Testability** with Go's testing framework

### Architecture

```
┌─────────────┐     ┌─────────────────────────────────┐
│   Sway WM   │────▶│      sway-cycle (Go CLI)        │
│  (IPC)      │◀────│                                  │
└─────────────┘     │  ┌─────────────────────────┐    │
                    │  │     IPC Manager         │    │
                    │  │  - Tree caching         │    │
                    │  │  - Event subscription   │    │
                    │  │  - Efficient parsing    │    │
                    │  └─────────────────────────┘    │
                    │                                  │
                    │  ┌─────────────────────────┐    │
                    │  │   Window Matcher        │    │
                    │  │  - Identifier detection  │    │
                    │  │  - Filter application    │    │
                    │  │  - Window validation     │    │
                    │  └─────────────────────────┘    │
                    │                                  │
                    │  ┌─────────────────────────┐    │
                    │  │   Cycle Manager         │    │
                    │  │  - List management      │    │
                    │  │  - Direction control    │    │
                    │  │  - Focus operations     │    │
                    │  └─────────────────────────┘    │
                    └─────────────────────────────────┘
```

### Components

#### 1. IPC Manager
- Handles connection to Sway IPC socket
- Manages window tree queries with configurable caching
- Subscribes to window events for cache invalidation
- Efficient JSON parsing with stream processing

#### 2. Window Matcher
- Extracts application identifiers (`app_id`, `class`, `instance`)
- Supports regex patterns for flexible matching
- Applies filters (exclude scratchpad, hidden, minimized)
- Validates window state before operations

#### 3. Cycle Manager
- Maintains ordered list of matching windows
- Supports forward/backward cycling
- Handles window destruction (cleanup from list)
- Manages focus operations with error handling

#### 4. Command Handler
- `cycle-forward`: Cycle to next window of same app (default)
- `cycle-backward`: Cycle to previous window of same app
- `list`: Show all windows matching current app
- `status`: Show current position and window count
- `daemon`: Run as background service for enhanced features

### Data Structures

```go
type WindowInfo struct {
    ID        int64     `json:"id"`
    AppID     string    `json:"app_id"`
    Class     string    `json:"class"`
    Instance  string    `json:"instance"`
    Name      string    `json:"name"`
    Workspace string    `json:"workspace"`
    Focused   bool      `json:"focused"`
    Type      string    `json:"type"` // "con", "floating_con"
    Scratchpad string   `json:"scratchpad_state"` // "none", "fresh", "hidden"
}

type CycleState struct {
    CurrentAppID string       `json:"current_app_id"`
    Windows      []WindowInfo `json:"windows"`
    CurrentIndex int          `json:"current_index"`
    LastUpdate   time.Time    `json:"last_update"`
}

type Config struct {
    // Storage configuration
    DBPath             string   `yaml:"db_path"`           // BoltDB database path
    
    // Matching configuration
    IdentifierPriority []string `yaml:"identifier_priority"` // ["app_id", "class", "instance"]
    ExcludeScratchpad  bool     `yaml:"exclude_scratchpad"`  // default: true
    ExcludeMinimized   bool     `yaml:"exclude_minimized"`   // default: true
    
    // Cycle behavior
    WrapAround         bool     `yaml:"wrap_around"`         // default: true
    MaintainOrder      bool     `yaml:"maintain_order"`      // maintain MRU order
    
    // Performance
    CacheTTL           duration `yaml:"cache_ttl"`           // Tree cache TTL
    MaxWindowsPerApp   int      `yaml:"max_windows_per_app"` // Limit cycling list
    
    // Application-specific rules
    AppRules           map[string]AppRule `yaml:"app_rules"`
}

type AppRule struct {
    MatchBy      string   `yaml:"match_by"`      // Override identifier priority
    ExcludeRegex string   `yaml:"exclude_regex"` // Exclude windows matching pattern
    GroupBy      string   `yaml:"group_by"`      // "workspace", "monitor", "none"
}
```

### Storage with BoltDB

The cycle system needs minimal state: window IDs per application and current position. BoltDB provides reliable storage without filesystem race conditions.

#### Simple Schema

```go
// Single bucket storing app -> window IDs + current index
const bucketCycles = "cycles"

type CycleStorage struct {
    db *bolt.DB
}

// Store window list for an app
func (c *CycleStorage) UpdateAppWindows(appID string, windowIDs []int64) error {
    return c.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte(bucketCycles))
        
        // Store as: appID + ":ids" -> binary window IDs
        //           appID + ":idx" -> current index (4 bytes)
        idKey := appID + ":ids"
        idxKey := appID + ":idx"
        
        // Convert IDs to binary
        buf := make([]byte, 0, len(windowIDs)*8)
        for _, id := range windowIDs {
            var idBytes [8]byte
            binary.LittleEndian.PutUint64(idBytes[:], uint64(id))
            buf = append(buf, idBytes[:]...)
        }
        
        // Store IDs
        if err := bucket.Put([]byte(idKey), buf); err != nil {
            return err
        }
        
        // Store current index (0 by default when updating list)
        var idxBytes [4]byte
        binary.LittleEndian.PutUint32(idxBytes[:], 0)
        return bucket.Put([]byte(idxKey), idxBytes[:])
    })
}

// Get next window ID for cycling
func (c *CycleStorage) GetNextWindow(appID string, forward bool) (int64, error) {
    var nextID int64
    err := c.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte(bucketCycles))
        
        // Read window IDs
        idKey := appID + ":ids"
        data := bucket.Get([]byte(idKey))
        if data == nil || len(data) == 0 {
            return fmt.Errorf("no windows for app %s", appID)
        }
        
        // Parse IDs
        var ids []int64
        for i := 0; i < len(data); i += 8 {
            ids = append(ids, int64(binary.LittleEndian.Uint64(data[i:i+8])))
        }
        
        // Read current index
        idxKey := appID + ":idx"
        idxData := bucket.Get([]byte(idxKey))
        currentIdx := 0
        if idxData != nil && len(idxData) >= 4 {
            currentIdx = int(binary.LittleEndian.Uint32(idxData))
        }
        
        // Calculate next index
        var nextIdx int
        if forward {
            nextIdx = (currentIdx + 1) % len(ids)
        } else {
            nextIdx = (currentIdx - 1 + len(ids)) % len(ids)
        }
        
        // Update stored index
        var newIdxBytes [4]byte
        binary.LittleEndian.PutUint32(newIdxBytes[:], uint32(nextIdx))
        if err := bucket.Put([]byte(idxKey), newIdxBytes[:]); err != nil {
            return err
        }
        
        nextID = ids[nextIdx]
        return nil
    })
    return nextID, err
}
```

#### File Location
```
~/.local/state/sway-cycle.bolt  # Separate database for simplicity
```

### API Design

```go
// Core interface
type CycleManager interface {
    // Window matching
    GetCurrentAppIdentifier() (string, string, error) // identifier, type
    FindMatchingWindows(identifier, identifierType string) ([]WindowInfo, error)
    
    // Cycling operations
    CycleForward() (WindowInfo, error)
    CycleBackward() (WindowInfo, error)
    JumpToIndex(index int) (WindowInfo, error)
    
    // State management
    GetCurrentState() (*CycleState, error)
    ClearState() error
    
    // Configuration
    UpdateConfig(config Config) error
}

// IPC interface  
type IPCManager interface {
    GetTree(cached bool) (*Tree, error)
    GetFocusedWindow() (*WindowInfo, error)
    FocusWindow(id int64) error
    SubscribeToEvents(events []string) (chan Event, error)
    Close() error
}

// Matcher interface
type WindowMatcher interface {
    ExtractIdentifier(window WindowInfo) (string, string) // identifier, type
    MatchWindows(windows []WindowInfo, identifier, identifierType string) []WindowInfo
    FilterWindows(windows []WindowInfo, filters FilterOptions) []WindowInfo
}
```

## Enhanced Features

### 1. Intelligent Window Grouping
- **Workspace-aware**: Option to cycle only within current workspace
- **Monitor-aware**: Cycle across monitors or within current monitor
- **MRU ordering**: Maintain most-recently-used order within app

### 2. Advanced Matching
- **Regex patterns**: Match windows by title patterns
- **Multi-criteria**: Combine app_id, class, and title for precise matching
- **Custom rules**: Application-specific matching rules

### 3. Performance Optimizations
- **Tree caching**: Configurable cache duration
- **Incremental updates**: Subscribe to window events
- **Background refresh**: Update cache in background
- **Connection pooling**: Reuse IPC connections

### 4. User Experience
- **Visual feedback**: Optional OSD notification of window position
- **Audible feedback**: Sound on wrap-around
- **Keyboard shortcuts**: Separate forward/backward shortcuts
- **Mouse integration**: Optional mouse wheel cycling

## Implementation Plan

### Phase 1: Core Library
1. **IPC client** with basic tree querying
2. **Window matching** with identifier extraction
3. **BoltDB storage** for cycle state persistence
4. **Basic cycling** (forward only)
5. **Simple CLI** with `cycle` command

### Phase 2: Enhanced Features
1. **Bidirectional cycling** (forward/backward)
2. **Configuration file** support
3. **Tree caching** with TTL
4. **Structured logging** and metrics

### Phase 3: Advanced Features
1. **Event subscription** for cache invalidation
2. **Application rules** with regex patterns
3. **Visual feedback** integration
4. **Systemd service** for daemon mode

### Phase 4: Integration
1. **Performance optimization**
2. **Comprehensive testing**
3. **Documentation**
4. **Migration from bash script**

## Configuration Examples

### Sway Config
```bash
# Basic forward cycling (default)
bindsym $mod+grave exec sway-cycle cycle

# Forward/backward cycling
bindsym $mod+grave exec sway-cycle cycle-forward
bindsym $mod+Shift+grave exec sway-cycle cycle-backward

# Daemon mode for enhanced performance
exec_always sway-cycle daemon --config ~/.config/sway-cycle.yaml
```

### Config File (`~/.config/sway-cycle.yaml`)
```yaml
# Storage configuration
db_path: ~/.local/state/sway-state.bolt

# Matching configuration
identifier_priority: ["app_id", "class", "instance"]
exclude_scratchpad: true
exclude_minimized: true

# Cycle behavior
wrap_around: true
maintain_order: true  # MRU ordering
cycle_within_workspace: false  # Cycle across all workspaces

# Performance
cache_ttl: "500ms"  # Tree cache time-to-live

# Application-specific rules
app_rules:
  "brave-browser":
    match_by: "app_id"
    group_by: "workspace"  # Cycle within workspace only
    
  "Alacritty":
    match_by: "app_id"
    exclude_regex: "^terminal-scratchpad$"  # Exclude scratchpad terminals
    
  "Code":
    match_by: "class"
    group_by: "monitor"  # Cycle within current monitor only

# Logging
log_level: "info"
log_file: "~/.local/state/sway-cycle.log"
```

### Systemd Service (`~/.config/systemd/user/sway-cycle.service`)
```ini
[Unit]
Description=Sway Window Cycling Daemon
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/sway-cycle daemon --config %h/.config/sway-cycle.yaml
Restart=on-failure
RestartSec=5
Environment="DISPLAY=:0"
Environment="WAYLAND_DISPLAY=wayland-1"

[Install]
WantedBy=default.target
```

## Migration from Bash

### Step 1: Coexistence
- Install Go binary as `sway-cycle-go`
- Update Sway config to use Go binary
- Keep bash script as fallback
- Compare behavior

### Step 2: Feature Parity
- Implement all bash script features
- Add backward compatibility for debug mode
- Ensure same window matching logic

### Step 3: Enhanced Operation
- Enable caching for better performance
- Add bidirectional cycling
- Introduce configuration file

### Step 4: Full Migration
- Remove bash script from startup
- Update documentation
- Clean up old bindings

## Performance Considerations

1. **Tree caching**: Configurable TTL balancing freshness vs performance
2. **Connection reuse**: Pool IPC connections to avoid setup overhead
3. **Selective parsing**: Parse only necessary window fields
4. **Background updates**: Update cache in background thread
5. **Memory limits**: Limit cached window data

## Testing Strategy

1. **Unit tests**: Window matching, cycle logic
2. **Integration tests**: IPC communication with mock
3. **Performance tests**: Tree parsing, cache efficiency
4. **End-to-end tests**: Full cycle in test environment
5. **Compatibility tests**: Ensure same behavior as bash script

## Monitoring and Observability

1. **Structured logs**: JSON format with log levels
2. **Prometheus metrics**:
   - `sway_cycle_operations_total` (counter)
   - `sway_cycle_cache_hits` (counter)
   - `sway_cycle_latency_seconds` (histogram)
   - `sway_cycle_window_count` (gauge)
3. **Health checks**: IPC connectivity, cache status
4. **Debug endpoints**: HTTP/Unix socket for troubleshooting

## Open Questions

1. Should cycling order be MRU or based on window creation time?
2. How to handle windows with same identifier but different workspaces?
3. Should we support cycling across applications (like Alt+Tab)?
4. How to integrate with existing marks system?
5. Should we support mouse wheel cycling as alternative input?

## Success Metrics

1. **Performance**: <50ms latency for cycle operations
2. **Reliability**: 99.9% successful focus operations
3. **Memory**: <20MB RSS for daemon mode
4. **CPU**: <0.5% average usage
5. **User satisfaction**: Natural feel similar to macOS behavior

## Next Steps

1. [ ] Create Go project structure
2. [ ] Implement basic IPC client
3. [ ] Create window matching logic
4. [ ] Implement forward cycling
5. [ ] Add CLI interface
6. [ ] Add configuration support
7. [ ] Implement backward cycling
8. [ ] Add caching layer
9. [ ] Create test suite
10. [ ] Write documentation