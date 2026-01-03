# Quick Start Guide

Get up and running with sway-compat in 5 minutes.

## TL;DR

```bash
# Build
git clone https://github.com/cristianoliveira/sway-compat
cd sway-compat
make build

# Install
sudo cp sway-compat /usr/local/bin/

# Add to Sway config
echo 'bindsym $mod+grave exec sway-compat cycle-forward' >> ~/.config/sway/config
echo 'bindsym $mod+Shift+grave exec sway-compat cycle-backward' >> ~/.config/sway/config

# Reload Sway
swaymsg reload
```

## What You Get

### Application Window Cycling (✅ Works Now)

**The Problem:** You have 5 Firefox windows open across different workspaces. Sway's default behavior makes it tedious to navigate between them.

**The Solution:** Press `$mod+\`` (usually `Super+\``) to cycle through just your Firefox windows, ignoring all other apps.

**Try It:**
1. Open 2-3 windows of the same application (e.g., multiple terminals)
2. Press `$mod+\`` repeatedly
3. Watch it cycle through only those windows

## Step-by-Step Setup

### 1. Build the Binary

```bash
git clone https://github.com/cristianoliveira/sway-compat
cd sway-compat
make build
```

This creates a `sway-compat` binary in the current directory.

### 2. Test It Works

Before installing, test it manually:

```bash
# Open 2-3 windows of the same app (like Firefox)
# Then run:
./sway-compat cycle-forward
```

If it switches to another window of the same app, it's working!

### 3. Install the Binary

Choose one:

**Option A: System-wide** (requires sudo)
```bash
sudo cp sway-compat /usr/local/bin/
```

**Option B: User-only** (no sudo)
```bash
mkdir -p ~/.local/bin
cp sway-compat ~/.local/bin/
# Make sure ~/.local/bin is in your PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

### 4. Configure Sway

Edit `~/.config/sway/config` and add:

```bash
# Application window cycling (like macOS)
bindsym $mod+grave exec sway-compat cycle-forward
bindsym $mod+Shift+grave exec sway-compat cycle-backward
```

**Note:** `grave` is the backtick key (`` ` ``), usually above Tab.

### 5. Reload Sway

```bash
swaymsg reload
```

## Usage Examples

### Example 1: Multiple Browser Windows

**Setup:**
- Workspace 1: Firefox window A
- Workspace 2: Firefox window B
- Workspace 3: Terminal
- Workspace 4: Firefox window C

**Usage:**
1. Focus any Firefox window
2. Press `$mod+\`` → goes to next Firefox window (could be on different workspace)
3. Press `$mod+\`` again → goes to third Firefox window
4. Press `$mod+\`` again → wraps back to first Firefox window
5. Terminal is never included because it's a different app

### Example 2: Multiple Terminals

**Setup:**
- 5 terminal windows scattered across workspaces
- You're editing code in one

**Usage:**
1. Press `$mod+\`` to quickly jump between terminals
2. Find the one with your server logs
3. Press `$mod+Shift+\`` to go back to the previous terminal

### Example 3: Mixed Apps

**Setup:**
- 3 Code editor windows
- 2 Firefox windows
- 1 Terminal

**When focused on Code:**
- `$mod+\`` only cycles through the 3 Code windows

**When focused on Firefox:**
- `$mod+\`` only cycles through the 2 Firefox windows

## Keybinding Reference

| Key Combo | What It Does | macOS Equivalent |
|-----------|--------------|------------------|
| `$mod+\`` | Cycle forward through app windows | `Cmd+\`` |
| `$mod+Shift+\`` | Cycle backward through app windows | `Cmd+Shift+\`` |

## Troubleshooting

**Q: It says "only one window found"**
- You only have one window of that app open
- Open another window of the same app to cycle between them

**Q: Wrong windows are being grouped together**
- Different versions of an app might have different app IDs
- Check with: `swaymsg -t get_tree | grep app_id`

**Q: It's not working at all**
- Make sure you ran `swaymsg reload` after editing the config
- Check the binary is in your PATH: `which sway-compat`
- Try running manually: `sway-compat cycle-forward`

**Q: I want to exclude certain windows**
- Configuration support is planned for future releases
- Currently uses sensible defaults (excludes scratchpad)

## What's Next?

- Check out the full [README.md](README.md) for detailed documentation
- Read [docs/CYCLE_USAGE.md](docs/CYCLE_USAGE.md) for advanced usage
- Explore the design docs in `docs/` to understand how it works

## Coming Soon

- **Window Focus Stack** (`Alt+Tab` style switching)
- **Configuration file** for customizing behavior
- **Per-app rules** for fine-grained control

Enjoy your improved Sway experience! 🚀
