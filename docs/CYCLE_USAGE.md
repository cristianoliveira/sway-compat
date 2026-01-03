# Window Cycling Usage Guide

The window cycling feature allows you to cycle through windows of the same application, similar to macOS `Cmd+\`` behavior.

## How It Works

1. **Identifier Matching**: The cycle manager identifies windows belonging to the same application by matching their `app_id`, `class`, or `instance` fields (in that priority order).

2. **Window Filtering**: By default, scratchpad windows are excluded from cycling. You can customize this behavior through configuration.

3. **Circular Cycling**: When you reach the last window and cycle forward, it wraps around to the first window (and vice versa for backward cycling).

## Commands

### cycle-forward
Cycles to the next window of the same application:
```bash
sway-compat cycle-forward
```

### cycle-backward
Cycles to the previous window of the same application:
```bash
sway-compat cycle-backward
```

### cycle
Shorthand for `cycle-forward`:
```bash
sway-compat cycle
```

## Sway Configuration

Add these bindings to your `~/.config/sway/config`:

```bash
# Cycle forward (like macOS Cmd+`)
bindsym $mod+grave exec sway-compat cycle-forward

# Cycle backward (like macOS Cmd+Shift+`)
bindsym $mod+Shift+grave exec sway-compat cycle-backward
```

After adding these bindings, reload your Sway configuration:
```bash
swaymsg reload
```

## Example Scenarios

### Multiple Browser Windows
If you have 3 Firefox windows open:
1. Press `$mod+\`` while focused on Firefox window 1
2. Focus moves to Firefox window 2
3. Press `$mod+\`` again
4. Focus moves to Firefox window 3
5. Press `$mod+\`` again
6. Focus wraps back to Firefox window 1

### Mixed Applications
If you have Firefox and Terminal windows open:
- Pressing `$mod+\`` while in Firefox only cycles through Firefox windows
- Pressing `$mod+\`` while in Terminal only cycles through Terminal windows
- The cycling is scoped to the current application

## Configuration Options

The cycle manager uses these default settings:

- **Identifier Priority**: `["app_id", "class", "instance"]` - tries to match by app_id first, then class, then instance
- **Exclude Scratchpad**: `true` - scratchpad windows are not included in cycling
- **Exclude Minimized**: `false` - minimized windows are included
- **Wrap Around**: `true` - cycling wraps from last to first window

## Troubleshooting

### "only one window found" error
This means there's only one window open for the current application. Cycling requires at least 2 windows.

### "no focused window found" error
Make sure a window is actually focused in Sway. Try clicking on a window first.

### "failed to connect to sway" error
Ensure you're running this tool inside a Sway session. The tool connects to Sway via the IPC socket.

## Implementation Details

The cycle feature is implemented in these components:

- **IPC Client** (`pkg/ipc/client.go`): Communicates with Sway to get window tree and focus windows
- **Window Matcher** (`pkg/cycle/matcher.go`): Identifies which windows belong to the same app
- **Cycle Manager** (`pkg/cycle/manager.go`): Orchestrates the cycling logic
- **Commands** (`cmd/cycle.go`): CLI interface for cycle operations

For more technical details, see the design document in `docs/sway-cycle-app.md`.
