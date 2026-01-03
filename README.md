# sway-compat

A collection of utilities for Sway window manager that provide macOS-like window cycling and stack management features.

## Features

### Window Cycling
Cycle through windows of the same application, similar to macOS `Cmd+`` behavior. This allows you to quickly switch between multiple windows of the same application (e.g., multiple browser windows or terminal instances).

### Window Stack
Alt+Tab style window switching with focus history. Maintains a stack of recently focused windows and allows quick toggling between current and previous windows.

## Installation

### From Source

```bash
git clone https://github.com/cristianoliveira/sway-compat
cd sway-compat
go build -o sway-compat
sudo cp sway-compat /usr/local/bin/
```

### Using Nix (if available)

```bash
nix build
```

## Usage

### Window Cycling

Cycle forward to the next window of the same application:
```bash
sway-compat cycle-forward
```

Cycle backward to the previous window of the same application:
```bash
sway-compat cycle-backward
```

Or use the shorthand:
```bash
sway-compat cycle  # defaults to cycle-forward
```

### Window Stack

Start the stack tracking daemon:
```bash
sway-compat stack daemon
```

Toggle between current and previous window:
```bash
sway-compat stack toggle
```

List all windows in the stack:
```bash
sway-compat stack list
```

Clear the window stack:
```bash
sway-compat stack clear
```

## Sway Configuration

Add these bindings to your Sway config (`~/.config/sway/config`):

```bash
# Window cycling (like macOS Cmd+`)
bindsym $mod+grave exec sway-compat cycle-forward
bindsym $mod+Shift+grave exec sway-compat cycle-backward

# Window stack (like Alt+Tab)
bindsym $mod+Tab exec sway-compat stack toggle

# Start stack daemon on Sway launch
exec_always sway-compat stack daemon
```

## Architecture

The project is organized into the following packages:

- `cmd/` - CLI commands using Cobra framework
- `pkg/ipc/` - Sway IPC communication
- `pkg/stack/` - Window focus stack management
- `pkg/cycle/` - Window cycling logic
- `pkg/storage/` - BoltDB storage for state persistence

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
│   ├── cycle.go           # Cycle commands
│   └── stack.go           # Stack commands
├── pkg/                    # Reusable packages
│   ├── ipc/               # Sway IPC client
│   ├── stack/             # Stack manager
│   ├── cycle/             # Cycle manager
│   └── storage/           # BoltDB storage
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
- [ ] Implement Sway IPC client
- [ ] Implement stack manager
- [ ] Implement cycle manager
- [ ] BoltDB storage implementation
- [ ] Configuration file support
- [ ] Performance optimizations
- [ ] Comprehensive test suite
- [ ] Documentation and examples

## Contributing

Contributions are welcome! Please read the design documents in the `docs/` directory to understand the architecture before contributing.

## License

MIT License - see LICENSE file for details

## Acknowledgments

This project aims to bring macOS-like window management features to Sway WM, enhancing productivity for users switching from macOS to Linux with Sway.
