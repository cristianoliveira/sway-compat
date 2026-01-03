package cycle

import (
	"fmt"

	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

// SimpleManager is a simple implementation of the Manager interface
type SimpleManager struct {
	ipcClient ipc.Manager
	matcher   *Matcher
	config    Config
}

// NewSimpleManager creates a new simple cycle manager
func NewSimpleManager(ipcClient ipc.Manager, config Config) *SimpleManager {
	return &SimpleManager{
		ipcClient: ipcClient,
		matcher:   NewMatcher(config),
		config:    config,
	}
}

// GetCurrentAppIdentifier returns the identifier of the currently focused app
func (m *SimpleManager) GetCurrentAppIdentifier() (string, string, error) {
	focused, err := m.ipcClient.GetFocusedWindow()
	if err != nil {
		return "", "", fmt.Errorf("failed to get focused window: %w", err)
	}

	if focused == nil {
		return "", "", fmt.Errorf("no focused window found")
	}

	identifier, identifierType := m.matcher.ExtractIdentifier(*focused)
	return identifier, identifierType, nil
}

// FindMatchingWindows finds all windows matching the given identifier
func (m *SimpleManager) FindMatchingWindows(identifier, identifierType string) ([]ipc.WindowInfo, error) {
	tree, err := m.ipcClient.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get window tree: %w", err)
	}

	// Get all windows from the tree
	allWindows := ipc.FindAllWindows(tree)

	// Filter out non-matching windows
	matchingWindows := m.matcher.MatchWindows(allWindows, identifier, identifierType)

	// Apply additional filters (scratchpad, minimized, etc.)
	filteredWindows := m.matcher.FilterWindows(matchingWindows)

	return filteredWindows, nil
}

// CycleForward cycles to the next window of the same application
func (m *SimpleManager) CycleForward() (ipc.WindowInfo, error) {
	// Get the current app identifier
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	// Find all matching windows
	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	if len(windows) == 0 {
		return ipc.WindowInfo{}, fmt.Errorf("no windows found for %s", identifier)
	}

	if len(windows) == 1 {
		// Only one window - nothing to cycle to, just return current window
		return windows[0], fmt.Errorf("only one %s window open (need at least 2 to cycle)", identifier)
	}

	// Find the current focused window index
	currentIndex := -1
	for i, window := range windows {
		if window.Focused {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return ipc.WindowInfo{}, fmt.Errorf("current window not found in matching windows")
	}

	// Calculate next index
	nextIndex := (currentIndex + 1) % len(windows)

	// If wrap around is disabled and we're at the end, don't cycle
	if !m.config.WrapAround && nextIndex == 0 && currentIndex == len(windows)-1 {
		return ipc.WindowInfo{}, fmt.Errorf("at last window and wrap around is disabled")
	}

	nextWindow := windows[nextIndex]

	// Focus the next window
	if err := m.ipcClient.FocusWindow(nextWindow.ID); err != nil {
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	return nextWindow, nil
}

// CycleBackward cycles to the previous window of the same application
func (m *SimpleManager) CycleBackward() (ipc.WindowInfo, error) {
	// Get the current app identifier
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	// Find all matching windows
	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	if len(windows) == 0 {
		return ipc.WindowInfo{}, fmt.Errorf("no windows found for %s", identifier)
	}

	if len(windows) == 1 {
		// Only one window - nothing to cycle to, just return current window
		return windows[0], fmt.Errorf("only one %s window open (need at least 2 to cycle)", identifier)
	}

	// Find the current focused window index
	currentIndex := -1
	for i, window := range windows {
		if window.Focused {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return ipc.WindowInfo{}, fmt.Errorf("current window not found in matching windows")
	}

	// Calculate previous index
	prevIndex := (currentIndex - 1 + len(windows)) % len(windows)

	// If wrap around is disabled and we're at the beginning, don't cycle
	if !m.config.WrapAround && prevIndex == len(windows)-1 && currentIndex == 0 {
		return ipc.WindowInfo{}, fmt.Errorf("at first window and wrap around is disabled")
	}

	prevWindow := windows[prevIndex]

	// Focus the previous window
	if err := m.ipcClient.FocusWindow(prevWindow.ID); err != nil {
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	return prevWindow, nil
}

// JumpToIndex jumps to a specific window by index
func (m *SimpleManager) JumpToIndex(index int) (ipc.WindowInfo, error) {
	// Get the current app identifier
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	// Find all matching windows
	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	if index < 0 || index >= len(windows) {
		return ipc.WindowInfo{}, fmt.Errorf("index %d out of range (0-%d)", index, len(windows)-1)
	}

	targetWindow := windows[index]

	// Focus the target window
	if err := m.ipcClient.FocusWindow(targetWindow.ID); err != nil {
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	return targetWindow, nil
}

// GetCurrentState returns the current cycle state
func (m *SimpleManager) GetCurrentState() (*State, error) {
	// Get the current app identifier
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return nil, err
	}

	// Find all matching windows
	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return nil, err
	}

	// Find current index
	currentIndex := -1
	for i, window := range windows {
		if window.Focused {
			currentIndex = i
			break
		}
	}

	return &State{
		CurrentAppID: fmt.Sprintf("%s (%s)", identifier, identifierType),
		Windows:      windows,
		CurrentIndex: currentIndex,
	}, nil
}

// ClearState clears the current cycle state (no-op for simple manager)
func (m *SimpleManager) ClearState() error {
	// Simple manager doesn't maintain state, so nothing to clear
	return nil
}

// UpdateConfig updates the cycle manager configuration
func (m *SimpleManager) UpdateConfig(config Config) error {
	m.config = config
	m.matcher = NewMatcher(config)
	return nil
}
