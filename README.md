# sway-compat

macOS-style application window cycling and MRU-style focus stack for the Sway tiling window manager.
The idea of this project is to align some UX that is only possible in macOS with AeroSpace, so I can have both systems NixOS and macOS with similar UX.

## Summary

- [Sway Compatibility](#sway-compatibility)
- [Features](#features)
- [Basic Usage](#basic-usage)
  - [Application Window Cycling](#application-window-cycling)
  - [Window Focus Stack](#window-focus-stack)
  - [Sway Config Examples](#sway-config-examples)
- [Installation](#installation)
- [How does it work?](#how-does-it-work)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [License](#license)

## Sway Compatibility

- Works with Sway on Wayland with IPC enabled (default on Sway sessions).
- Tested on recent Sway releases; please report issues you hit with specific versions.

**Beta**: I use this daily, but there may be breaking changes before a 1.0.0 release. Please report bugs and ideas in the [issues](https://github.com/cristianoliveira/sway-compat/issues).

## Features

- **Application Window Cycling** (Cmd+apostrophe - `mod+``): cycle through only the windows of the currently focused application.
- **Window Focus Stack** (`Alt+Tab`-style): toggle between the two most recent windows or walk a MRU stack.
- **Scratchpad Focus Handling**: macOS/AeroSpace-style auto-hide scratchpad when focusing tiling windows.
- Excludes scratchpad windows by default, uses Sway IPC directly, and provides a daemon for instant MRU toggling.

## Basic Usage

New to `sway-compat`? Start with the 5-minute [`QUICKSTART.md`](QUICKSTART.md).

### Application Window Cycling

Cycle through windows of the currently focused application:
```bash
sway-compat cycle-forward
```

Cycle in reverse:
```bash
sway-compat cycle-backward
```

`sway-compat cycle` defaults to forward cycling. Windows are grouped by `app_id`, `class`, or `instance`.

### Window Focus Stack

Start the daemon (tracks focus history and exposes IPC):
```bash
sway-compat daemon
```

Toggle between your two most recent windows:
```bash
sway-compat stack toggle
```

Inspect or reset the stack:
```bash
sway-compat stack list
sway-compat stack clear
```

Useful flags:
- `--db-path <path>`: override stack persistence location (default `~/.local/state/sway-compat-stack.json`)
- `--stack-size <n>`: limit tracked windows (default `20`)
- `--exclude <apps>` / `--include-only <apps>`: comma-separated filters for app IDs

### Scratchpad Focus Handling

Replicate macOS/AeroSpace behavior where scratchpad windows automatically hide when you focus a tiling window in the same workspace:

```bash
# Enable scratchpad focus handling with daemon
sway-compat daemon --scratchpad-focus

# Optional configuration
sway-compat daemon --scratchpad-focus \
  --scratchpad-hide-action hide-scratchpad \
  --scratchpad-workspace .scratchpad \
  --scratchpad-debounce 200
```

**Behavior:**
- When a floating scratchpad window loses focus to a tiling window in the same workspace
- The tiling window is brought to front (re-focused)
- The scratchpad window is moved back to the scratchpad workspace (if configured)

**Flags:**
- `--scratchpad-focus`: enable the feature (default: `false`)
- `--scratchpad-hide-action`: `hide-scratchpad` (move to scratchpad) or `noop` (default: `hide-scratchpad`)
- `--scratchpad-workspace`: scratchpad workspace name (default: `.scratchpad`)
- `--scratchpad-debounce`: debounce interval in milliseconds to avoid rapid hide/show (default: `200`)

For detailed design and implementation notes, see [`docs/sway-scratchpad-focus.md`](docs/sway-scratchpad-focus.md).

### Sway Config Examples

Add to `~/.config/sway/config`:
```bash
# App window cycling (macOS Cmd+`)
bindsym $mod+grave exec sway-compat cycle-forward
bindsym $mod+Shift+grave exec sway-compat cycle-backward

# Focus stack (Alt+Tab-style)
bindsym $mod+Tab exec sway-compat stack toggle
exec_always sway-compat daemon

# Optional: Scratchpad focus handling (macOS/AeroSpace-style)
# exec_always sway-compat daemon --scratchpad-focus

# Optional filters for the daemon
# exec_always sway-compat daemon --exclude waybar,swaylock
# exec_always sway-compat daemon --include-only firefox,Alacritty,code
```

Reload Sway after updating the config:
```bash
swaymsg reload
```

## Installation

### Nix

```bash
nix build
```

### Go

```bash
go install github.com/cristianoliveira/sway-compat@latest
```

### From Source

```bash
git clone https://github.com/cristianoliveira/sway-compat
cd sway-compat
make build
sudo make install      # installs to /usr/local/bin
# or:
make install-local     # installs to ~/.local/bin
```

Verify:
```bash
sway-compat --version
sway-compat --help
```

## How does it work?

- Uses Sway IPC directly to identify focused windows and enumerate matches by `app_id`, `class`, or `instance`.
- Cycling commands run as one-off CLI calls (scratchpad windows excluded by default).
- The daemon listens for focus events, maintains an in-memory MRU stack, persists to JSON, and exposes a Unix socket for `stack` commands.
- Scratchpad focus handling monitors focus changes and automatically hides scratchpad windows when tiling windows are focused in the same workspace.

## Troubleshooting

- **"only one window found"**: open another window of the same app before cycling.
- **"no focused window found"**: focus a window in Sway, then retry.
- **"failed to connect to sway"**: ensure you are inside a Sway session with `SWAYSOCK` set.
- **"Not enough windows in stack to toggle"**: start the daemon (`sway-compat daemon`) and focus at least two windows.
- **"Failed to connect to daemon"**: ensure the daemon is running and that the socket exists in `$XDG_RUNTIME_DIR` or `/tmp/sway-compat-$UID.sock`.

Enable debug logging:
```bash
SWAY_COMPAT_LOGS_LEVEL=DEBUG SWAY_COMPAT_LOGS_PATH=/tmp/sway-compat.log sway-compat cycle-forward
```
or set those variables in your shell/Sway config for persistent logs. Inspect with `tail -f /tmp/sway-compat.log`.

## Development

```bash
go test ./...
go build -o sway-compat
```

See `docs/` for design notes and deeper guides on cycling and stack behavior.

## License

MIT License – see `LICENSE`.
