# sway-compat

A collection of window management utilities for Sway WM that brings familiar behaviors from other operating systems and window managers. This tool enhances Sway with productivity features that power users expect, making the transition from macOS or other desktop environments smoother.

> 🚀 **New to sway-compat?** Check out the [Quick Start Guide](QUICKSTART.md) for a 5-minute setup!

## Features Overview

This project implements two powerful window management behaviors that are built-in to other window managers but missing from Sway:

### 1. Application Window Cycling (✅ Implemented)
**Origin: macOS** (`Cmd+\``)

Cycle through multiple windows of the **same application**. This is the behavior you get on macOS when pressing `Cmd+\`` (backtick) - it cycles only through windows of the currently focused application, not all windows.

**Why it's useful:**
- You have 3 browser windows open → quickly cycle between just those 3 browsers
- You have 5 terminal windows → navigate between terminals without seeing other apps
- Keeps you focused within your current application context

**How it works:**
- Identifies windows by `app_id`, `class`, or `instance`
- Only cycles through windows of the same application
- Excludes scratchpad windows by default
- Circular navigation (wraps around from last to first)

### 2. Window Focus Stack (✅ Implemented)
**Origin: Most desktop environments** (Windows, KDE, GNOME, etc.)

Traditional `Alt+Tab` behavior that switches between windows based on **recency of focus** (MRU - Most Recently Used). This maintains a stack of your window focus history and lets you toggle back and forth.

**Why it's useful:**
- Quickly toggle between your two most recent windows (e.g., browser ↔ terminal)
- Access recently used windows in order
- More intuitive than Sway's workspace-based navigation for rapid context switching

**How it works:**
- Background daemon tracks all window focus changes and runs an IPC server
- Commands communicate with daemon via Unix domain socket (IPC)
- Daemon maintains stack in memory (fast) and persists to JSON file (survives restarts)
- Single source of truth: daemon owns all state, no race conditions
- Toggle command switches to previous window instantly via IPC
- Window validation ensures closed windows are skipped
- Configurable exclusion/inclusion filters for app IDs

## Installation

### Prerequisites
- Go 1.21 or later
- Sway window manager
- Access to Sway IPC socket (automatically available in Sway sessions)

### From Source

```bash
# Clone the repository
git clone https://github.com/cristianoliveira/sway-compat
cd sway-compat

# Build the binary
make build

# Install to /usr/local/bin (optional)
sudo make install

# Or install to ~/.local/bin (no sudo needed)
mkdir -p ~/.local/bin
cp sway-compat ~/.local/bin/
# Make sure ~/.local/bin is in your PATH
```

### Using Nix (if available)

```bash
nix build
```

### Verify Installation

```bash
sway-compat --version
sway-compat --help
```

## Usage Guide

### Application Window Cycling (macOS-style)

This feature lets you cycle through windows of the **same application only**.

#### Commands

**Cycle forward** (to next window):
```bash
sway-compat cycle-forward
```

**Cycle backward** (to previous window):
```bash
sway-compat cycle-backward
```

**Shorthand** (defaults to forward):
```bash
sway-compat cycle
```

#### Example Scenario

Imagine you have these windows open:
- Firefox (window 1)
- Firefox (window 2)
- Terminal
- Firefox (window 3)

When focused on any Firefox window and you run `cycle-forward`:
1. First press → moves to Firefox window 2
2. Second press → moves to Firefox window 3
3. Third press → wraps to Firefox window 1
4. The Terminal is never included because it's a different app

#### Sway Configuration

Add to `~/.config/sway/config`:

```bash
# Application window cycling (like macOS Cmd+`)
bindsym $mod+grave exec sway-compat cycle-forward
bindsym $mod+Shift+grave exec sway-compat cycle-backward
```

Common key mappings:
- `$mod+grave` - Usually `Super+`` (backtick key)
- Similar to macOS `Cmd+`` behavior
- Use `Shift` modifier for reverse direction

After adding, reload Sway:
```bash
swaymsg reload
```

#### Visual Reference

macOS equivalent:
```
macOS:  Cmd+`           → cycle through Safari windows only
Sway:   $mod+grave      → cycle through Firefox windows only
```

### Window Focus Stack (Alt+Tab-style)

This feature maintains a history of focused windows and lets you quickly toggle between them based on recency of use.

#### Commands

**Start the daemon** (tracks focus history and runs IPC server):
```bash
sway-compat daemon
```

The daemon supports several flags:
- `--db-path <path>` - Custom storage location (default: `~/.local/state/sway-compat-stack.json`)
- `--stack-size <n>` - Maximum windows to track (default: 20)
- `--exclude <apps>` - Comma-separated app IDs to exclude (e.g., `waybar,swaylock`)
- `--include-only <apps>` - Only track these app IDs (comma-separated)

**Note:** The daemon must be running for stack commands (toggle/list/clear) to work, as they communicate with the daemon via IPC.

**Toggle** between current and previous window:
```bash
sway-compat stack toggle
```

**List** current stack:
```bash
sway-compat stack list
```

**Clear** the stack:
```bash
sway-compat stack clear
```

#### Example Scenario

Your window focus history:
1. Terminal (current)
2. Browser (previous)
3. Editor
4. File manager

When you run `stack toggle`:
- First press → switches to Browser
- Second press → switches back to Terminal
- Rapid toggling between your two most recent windows

#### Sway Configuration

Add to `~/.config/sway/config`:

```bash
# Window focus stack (like Alt+Tab)
bindsym $mod+Tab exec sway-compat stack toggle

# Start the daemon on Sway launch
exec_always sway-compat daemon

# Optional: Exclude certain apps from the stack
# exec_always sway-compat daemon --exclude waybar,swaylock

# Optional: Only track specific apps
# exec_always sway-compat daemon --include-only firefox,Alacritty,code
```

After adding, reload Sway:
```bash
swaymsg reload
```

#### Visual Reference

Traditional desktop equivalent:
```
Windows/Linux:  Alt+Tab         → switch to previous window
Sway:           $mod+Tab        → switch to previous window (when implemented)
```

## Quick Reference

### Behavior Comparison Table

| Feature | Origin | Traditional Keybinding | Sway Binding | Implementation Status |
|---------|--------|----------------------|--------------|---------------------|
| **Application Window Cycling** | macOS | `Cmd+\`` | `$mod+grave` | ✅ Implemented |
| **Reverse App Cycling** | macOS | `Cmd+Shift+\`` | `$mod+Shift+grave` | ✅ Implemented |
| **Focus Stack Toggle** | Windows/Linux | `Alt+Tab` | `$mod+Tab` | ✅ Implemented |

### What Each Feature Does

| You Want To... | Use This Feature | Command |
|----------------|-----------------|---------|
| Switch between 3 Firefox windows only | Application Cycling | `cycle-forward` |
| Go back to your previous window (any app) | Focus Stack | `stack toggle` |
| Navigate multiple terminals without seeing other apps | Application Cycling | `cycle-forward` |
| Quick toggle: Editor ↔ Browser | Focus Stack | `stack toggle` |

## Troubleshooting

### Application Window Cycling

**Error: "only one window found"**
- You only have one window of that application open
- Cycling requires at least 2 windows of the same app
- Solution: Open another window of the same application

**Error: "no focused window found"**
- No window is currently focused in Sway
- Solution: Click on a window to focus it, then try again

**Error: "failed to connect to sway"**
- Not running inside a Sway session
- The `SWAYSOCK` environment variable is not set
- Solution: Make sure you're running this inside Sway WM

**Wrong windows are being cycled**
- The tool matches by `app_id`, `class`, or `instance`
- Some apps may have different identifiers for different windows
- Solution: Check window properties with `swaymsg -t get_tree`

**Cycling doesn't include all windows**
- By default, scratchpad windows are excluded
- Future versions will support configuration to customize this

### Window Focus Stack

**Error: "Not enough windows in stack to toggle"**
- Stack has fewer than 2 windows
- The daemon may not be running
- Solution: Start the daemon with `sway-compat daemon`

**Error: "Failed to connect to daemon"**
- Daemon not running in background
- IPC socket doesn't exist
- Check if running: `ps aux | grep "sway-compat daemon"`
- Check socket: `ls -la $XDG_RUNTIME_DIR/sway-compat.sock` or `/tmp/sway-compat-$UID.sock`
- Solution: Add `exec_always sway-compat daemon` to Sway config

**Stack doesn't persist after restart**
- Storage file may not have write permissions
- Default location: `~/.local/state/sway-compat-stack.json`
- Solution: Check directory exists and is writable: `mkdir -p ~/.local/state`

**Wrong windows in stack**
- Some apps are being tracked that shouldn't be
- Solution: Use `--exclude` flag to filter out unwanted apps
- Example: `sway-compat daemon --exclude waybar,swaylock`

**Daemon uses too much memory**
- Stack size may be too large
- Solution: Reduce with `--stack-size` flag
- Example: `sway-compat daemon --stack-size 10`

### General Issues

**Command not found**
- Binary not in PATH
- Solution: Ensure `sway-compat` is installed to a directory in your `$PATH`
- Check with: `which sway-compat`

**Permission denied**
- Binary doesn't have execute permissions
- Solution: `chmod +x sway-compat`

### Debugging with Logs

If you encounter issues with `sway-compat`, you can enable detailed logging using environment variables:

- **SWAY_COMPAT_LOGS_PATH**: Set this to specify the path for log files. Default: `/tmp/sway-compat.log`
- **SWAY_COMPAT_LOGS_LEVEL**: Set the logging level. Default: `DISABLED`. Options: `DEBUG`, `INFO`, `WARN`, `ERROR`

#### Enable Debug Logging

**Temporary (for one command):**
```bash
SWAY_COMPAT_LOGS_LEVEL=DEBUG sway-compat cycle-forward
```

**Persistent (in your shell):**
```bash
export SWAY_COMPAT_LOGS_LEVEL=DEBUG
export SWAY_COMPAT_LOGS_PATH=/tmp/sway-compat.log
```

Add to `~/.bashrc` or `~/.zshrc` to make it permanent.

**In Sway config:**
```bash
# Enable debug logging for cycle commands
bindsym $mod+grave exec env SWAY_COMPAT_LOGS_LEVEL=DEBUG sway-compat cycle-forward

# Enable debug logging for the daemon
exec_always env SWAY_COMPAT_LOGS_LEVEL=DEBUG sway-compat daemon
```

#### View Logs

```bash
# Watch logs in real-time
tail -f /tmp/sway-compat.log

# View recent logs
cat /tmp/sway-compat.log
```

The logs will show:
- IPC connection status
- Window detection and matching
- Cycling operations with window IDs and names
- Stack daemon focus change events
- Window push/toggle operations
- Any errors encountered

#### Example Debug Output

**Cycle forward:**
```
time=2026-01-04T00:06:55.008+01:00 level=DEBUG msg="sway-compat starting"
time=2026-01-04T00:06:55.009+01:00 level=DEBUG msg="Connecting to Sway IPC"
time=2026-01-04T00:06:55.009+01:00 level=INFO msg="Successfully connected to Sway IPC"
time=2026-01-04T00:06:55.009+01:00 level=DEBUG msg="Getting focused window"
time=2026-01-04T00:06:55.009+01:00 level=DEBUG msg="Found focused window" id=7 name=Alacritty app_id=term-main
time=2026-01-04T00:06:55.010+01:00 level=DEBUG msg="Current app identifier" identifier=term-main type=app_id
time=2026-01-04T00:06:55.010+01:00 level=DEBUG msg="Found matching windows" count=3
time=2026-01-04T00:06:55.010+01:00 level=INFO msg="Cycling to next window" from_id=7 to_id=14 to_name="Alacritty"
```

**Daemon:**
```
time=2026-01-04T00:10:23.001+01:00 level=INFO msg="Starting sway-compat daemon"
time=2026-01-04T00:10:23.002+01:00 level=INFO msg="IPC server started" socket=/run/user/1000/sway-compat.sock
time=2026-01-04T00:10:23.003+01:00 level=INFO msg="Daemon started successfully" db_path=/home/user/.local/state/sway-compat-stack.json socket_path=/run/user/1000/sway-compat.sock stack_size=20
time=2026-01-04T00:10:25.120+01:00 level=DEBUG msg="Focus change detected" window_id=12345 window_name="Firefox" app_id=firefox
time=2026-01-04T00:10:25.121+01:00 level=DEBUG msg="Pushing window to stack" window_id=12345 window_name="Firefox" app_id=firefox
time=2026-01-04T00:10:25.122+01:00 level=INFO msg="Window pushed to stack" window_id=12345 stack_size=3
```

## Architecture

The project is organized into the following packages:

- `cmd/` - CLI commands using Cobra framework
  - `daemon.go` - Top-level daemon command (IPC server + event tracking)
  - `stack.go` - Stack commands that communicate via IPC
  - `cycle.go` - Cycle commands (direct Sway IPC)
- `pkg/ipc/` - IPC communication
  - Sway IPC client (using swayipc library)
  - Daemon IPC server (Unix domain socket)
  - Daemon IPC client (for stack commands)
- `pkg/stack/` - Window focus stack management (implemented)
- `pkg/cycle/` - Application window cycling logic (implemented)
- `pkg/storage/` - JSON file storage for state persistence (implemented)
- `internal/logger/` - Structured logging with configurable levels

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o sway-compat
```

### Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── daemon.go          # Daemon command (IPC server + event tracking)
│   ├── cycle.go           # Cycle commands
│   └── stack.go           # Stack commands (IPC clients)
├── pkg/                    # Reusable packages
│   ├── ipc/               # IPC communication
│   │   ├── client.go      # Sway IPC client
│   │   ├── server.go      # Daemon IPC server
│   │   ├── daemon_client.go # Daemon IPC client
│   │   └── protocol.go    # IPC protocol types
│   ├── stack/             # Stack manager
│   ├── cycle/             # Cycle manager
│   └── storage/           # JSON file storage
├── docs/                   # Documentation
│   ├── project-setup.md   # Development setup
│   ├── sway-cycle-app.md  # Cycle feature design
│   └── sway-stack.md      # Stack feature design
├── main.go                # Entry point
└── README.md              # This file
```

## Roadmap

- [x] Basic project setup
- [x] CLI framework with Cobra
- [x] Package structure and interfaces
- [x] Testing infrastructure
- [x] Implement Sway IPC client
- [x] Implement cycle manager (window cycling)
- [x] Window matching logic
- [x] Cycle commands (forward/backward)
- [x] Unit tests for cycle functionality
- [x] Structured logging with debug support
- [x] Implement stack manager (Alt+Tab behavior)
- [x] JSON file storage implementation
- [x] Event loop for window focus tracking
- [x] Stack daemon with signal handling
- [x] Stack commands (toggle/list/clear)
- [x] Comprehensive unit tests for stack
- [x] IPC architecture (daemon as server, commands as clients)
- [x] Unix domain socket IPC protocol
- [x] Top-level daemon command
- [ ] Configuration file support
- [ ] Performance optimizations
- [ ] Integration tests
- [ ] Advanced filtering and rules

## Contributing

Contributions are welcome! Please read the design documents in the `docs/` directory to understand the architecture before contributing.

### Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `make test`
4. Build: `make build`
5. Read the design docs in `docs/` before implementing features

## Related Projects

- [Sway](https://github.com/swaywm/sway) - The tiling Wayland compositor
- [swayipc](https://codeberg.org/scip/swayipc) - Go library for Sway IPC communication
- [i3-cycle](https://github.com/un-def/i3-cycle) - Similar cycling behavior for i3wm

## License

MIT License - see LICENSE file for details

## Acknowledgments

### Inspiration

This project brings familiar window management behaviors from other systems to Sway:

- **macOS** - Application window cycling (`Cmd+\``) behavior is the gold standard for managing multiple windows of the same app. macOS has had this built-in since the early days of Mac OS X.

- **Windows/Linux DEs** - Focus stack / Alt+Tab behavior has been a staple of desktop computing since Windows 95 and is present in KDE, GNOME, Xfce, and virtually every traditional desktop environment.

### Why These Features Matter

Sway is an excellent tiling window manager, but it's built around a workspace-centric model. While powerful, this can be less intuitive for users coming from:
- **macOS** where `Cmd+Tab` switches apps and `Cmd+\`` switches windows within an app
- **Traditional DEs** where `Alt+Tab` provides MRU (Most Recently Used) window switching

This tool bridges that gap, letting you enjoy Sway's tiling capabilities while keeping the window navigation patterns you're already familiar with.

### Credits

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [swayipc](https://codeberg.org/scip/swayipc) for Sway IPC communication
- Uses Unix domain sockets for daemon-command IPC
- Inspired by years of muscle memory from macOS and traditional desktop environments

## Support

If you find this tool useful:
- ⭐ Star the repository
- 🐛 Report bugs via GitHub Issues
- 💡 Suggest features via GitHub Discussions
- 🔀 Submit pull requests

For questions about usage, see the [docs/CYCLE_USAGE.md](docs/CYCLE_USAGE.md) guide.
