# Scratchpad Focus Handling — User Guide

> **Note**: This feature is now implemented in sway-compat v0.2.0+

## Overview

Replicate the macOS/AeroSpace behavior where, when a scratchpad (floating) window loses focus and another window in the same workspace gains focus, the non-floating window is brought to the front. If Sway cannot re-order stacking to place the tiling window above the floating one, fall back to sending the scratchpad window back to the scratchpad workspace.

## Quick Start

Enable scratchpad focus handling by adding this to your Sway config (`~/.config/sway/config`):

```bash
exec_always sway-compat daemon --scratchpad-focus
```

Or with custom configuration:

```bash
exec_always sway-compat daemon --scratchpad-focus \
  --scratchpad-hide-action hide-scratchpad \
  --scratchpad-workspace .scratchpad \
  --scratchpad-debounce 200
```

**Reload Sway** after updating the config:
```bash
swaymsg reload
```

## Usage

### Behavior

When enabled, the daemon will:
1. Detect when a floating scratchpad window loses focus to a tiling window in the same workspace
2. Attempt to bring the tiling window to front by re-focusing it
3. If configured with `--scratchpad-hide-action hide-scratchpad`, move the scratchpad window back to the scratchpad workspace

### Configuration Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--scratchpad-focus` | Enable scratchpad focus handling | `false` |
| `--scratchpad-hide-action` | Action when scratchpad loses focus: `hide-scratchpad` (move to scratchpad) or `noop` | `hide-scratchpad` |
| `--scratchpad-workspace` | Scratchpad workspace name | `.scratchpad` |
| `--scratchpad-debounce` | Debounce interval in milliseconds to avoid rapid hide/show thrash | `200` |

### Examples

**Basic usage:**
```bash
sway-compat daemon --scratchpad-focus
```

**Custom scratchpad workspace:**
```bash
sway-compat daemon --scratchpad-focus --scratchpad-workspace scratch
```

**Disable auto-hiding (only re-focus tiling window):**
```bash
sway-compat daemon --scratchpad-focus --scratchpad-hide-action noop
```

## Design Details

> The following sections describe the original design plan that has been implemented.

## Goals

- Detect when a floating scratchpad window loses focus to another window in the same workspace.
- Attempt to surface the newly focused tiling/non-floating window above the scratchpad.
- If Sway stacking cannot be adjusted, hide the scratchpad window by moving it back to the scratchpad workspace.
- Provide opt-in behavior with minimal configuration.

## Non-Goals

- Managing arbitrary floating windows that are not treated as scratchpads.
- Replacing existing scratchpad show/hide commands.
- Altering Sway focus policies globally.

## Assumptions and Constraints

- Sway floating windows remain above tiling by default; lowering a floating window may not be supported directly.
- Moving a window back to the scratchpad workspace is always possible (`[con_id=ID] move scratchpad`).
- We already depend on Sway IPC in the daemon; reuse that channel.

## Proposed Approach

1. **Event Source**  
   Extend the existing daemon (or create a small watcher) to subscribe to `window` events via Sway IPC (`focus`, `floating`, `move`).

2. **Scratchpad Detection**  
   - Treat a window as scratchpad if `floating == true` and `scratchpad_state != "none"` or if its workspace name matches the configured scratchpad workspace (default `.scratchpad`).  
   - Cache the current focused window ID and workspace.

3. **Focus Change Handler**  
   On focus event:
   - `prev` = last focused window, `curr` = new focused window.
   - If `prev` is a scratchpad window, `curr` is in the same workspace (or the workspace where `prev` was shown), and `curr` is non-floating:
     - Attempt to bring `curr` to front:
       - Option A: `swaymsg [con_id=curr] focus` (no-op if already focused, but re-applies focus).
       - Option B: If Sway exposes `focus tiling` or similar, issue that.
     - If floating still obscures content (cannot lower), then send `prev` back to scratchpad: `swaymsg [con_id=prev] move scratchpad`.

4. **Configuration Knobs**
   - Enable/disable feature (default: off to avoid surprises).
   - Choose fallback action: `hide-scratchpad` (default) or `noop`.
   - Scratchpad workspace name (default `.scratchpad`).
   - Debounce interval to avoid rapid hide/show thrash (e.g., 150–250ms).

5. **Failure Modes**
   - If IPC command fails, log debug message and do nothing further.
   - If multiple floating scratchpads are visible, only act on the one that lost focus.

## Event Flow

1. User shows a scratchpad window (floating) into workspace `W`.
2. User focuses a tiling window in `W`.
3. Daemon receives focus event (`prev` scratchpad floating, `curr` tiling in same workspace).
4. Daemon issues focus reaffirmation to `curr`; if configured or needed, issues `move scratchpad` to `prev`.

## Edge Cases

- **Workspace change**: If focus moves to a different workspace, skip action.
- **No tiling windows**: If only floating windows exist, skip.
- **Multiple monitors**: Use workspace matching, not output matching.
- **Scratchpad already hidden**: No action.
- **Non-scratchpad floating windows**: Ignore unless explicitly configured.

## Testing Plan

- Unit-test the focus handler decision matrix (prev/curr permutations: floating vs tiling, same workspace vs different).
- Integration tests (if available) using sway mock or recorded IPC streams.
- Manual checks:
  - Show scratchpad, click tiling window → scratchpad hides or tiling confirmed on top.
  - Switch workspace → scratchpad remains unaffected.
  - Multiple scratchpads visible → only the one that lost focus is acted on.

## Open Questions

- Does Sway expose a way to lower a floating window or adjust stacking? If not, hiding to scratchpad is the fallback.
- Should this run inside the existing stack daemon or as a separate watcher command (e.g., `sway-compat scratchpad-focus-daemon`)? Reuse the daemon if IPC load is acceptable.
- How to handle users who rely on floating overlays they do not want auto-hidden? Configuration defaulting to off mitigates this.

## Code Touch Points (when implementing)

- `cmd/daemon.go`: subscribe to Sway `window` events (focus/floating/move) and invoke the scratchpad focus handler.
- `pkg/ipc/*` (Sway IPC client/subscription layer): ensure window event payloads expose `con_id`, workspace, `floating`, and `scratchpad_state`.
- `pkg/stack/` (or a new module if you keep it separate): implement the focus-change decision logic, configuration toggles, and debounce. Reuse window classification helpers from cycle/stack if available.
- `internal/logger/`: add debug logs for focus transitions and actions (re-focus vs. move to scratchpad) and IPC errors.
- `main.go` / command wiring: surface a flag/config to enable this behavior (default off) and pass it into the daemon.
- Tests: add unit tests covering the decision matrix (prev/curr floating vs. tiling, same vs. different workspace, config toggles, fallback actions).***
