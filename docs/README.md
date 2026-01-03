# Documentation Index

Welcome to the sway-compat documentation! This directory contains detailed guides and design documents.

## For Users

### Getting Started
- **[Quick Start Guide](../QUICKSTART.md)** - Get up and running in 5 minutes
- **[Main README](../README.md)** - Full project overview and feature comparison

### Feature Guides
- **[Window Cycling Usage Guide](CYCLE_USAGE.md)** - Detailed guide for application window cycling (macOS-style)
  - How it works
  - Usage examples
  - Configuration options
  - Troubleshooting

### Configuration Examples
- **[examples/sway-config-cycle.conf](examples/sway-config-cycle.conf)** - Example Sway configuration for window cycling

## For Developers

### Design Documents
These documents outline the architecture and design decisions for each feature:

- **[sway-cycle-app.md](sway-cycle-app.md)** - Application window cycling system design
  - Current bash implementation analysis
  - Go CLI architecture
  - Data structures and APIs
  - Storage with BoltDB
  - Implementation phases

- **[sway-stack.md](sway-stack.md)** - Window focus stack system design
  - Window stack management (Alt+Tab behavior)
  - Event-driven architecture
  - State persistence
  - Daemon mode design

### Development Setup
- **[project-setup.md](project-setup.md)** - Development tools and setup
  - CLI framework (cobra-cli)
  - Testing tools (gotestsum)
  - Mock generation (mockgen)
  - Sway IPC integration (swayipc)

## Feature Comparison

| Feature | Status | Documentation |
|---------|--------|---------------|
| Application Window Cycling | ✅ Implemented | [CYCLE_USAGE.md](CYCLE_USAGE.md) |
| Window Focus Stack | ⏳ Planned | [sway-stack.md](sway-stack.md) |

## Reference Scripts

The `examples/` directory contains the original bash scripts that inspired this project:

- **sway-focus-tracker.sh** - Original bash daemon for tracking window focus
- **sway-stack.sh** - Original bash script for window stack toggling

These are being replaced by the Go implementation for better performance, reliability, and maintainability.

## Quick Links

### Understanding the Features

**Question:** What's the difference between cycle and stack?

**Answer:**
- **Cycle** = Navigate between windows of the **same app** (like Firefox window 1 → 2 → 3)
- **Stack** = Navigate between **any windows** based on recency (Browser → Terminal → Editor)

**Question:** Which window managers have these features built-in?

**Answer:**
- **Application Cycling**: macOS (Cmd+\`), GNOME (on same workspace)
- **Focus Stack**: Windows (Alt+Tab), KDE (Alt+Tab), GNOME (Alt+Tab), most traditional DEs

**Question:** Why isn't this built into Sway?

**Answer:** Sway is workspace-centric by design. These features are application/window-centric, which is a different paradigm. This tool bridges that gap.

## Contributing

If you want to contribute:

1. Read the [main README](../README.md) first
2. Review the design docs for the feature you want to work on
3. Check the [project setup guide](project-setup.md)
4. Look at existing implementations in `pkg/` for patterns

## Getting Help

- **Usage questions**: Check [CYCLE_USAGE.md](CYCLE_USAGE.md) or [QUICKSTART.md](../QUICKSTART.md)
- **Bug reports**: GitHub Issues
- **Feature requests**: GitHub Discussions
- **Development questions**: Read the design docs, then ask in GitHub Discussions
