package cycle

import (
	"fmt"

	"github.com/cristianoliveira/sway-compat/internal/logger"
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

	allWindows := ipc.FindAllWindows(tree)
	matchingWindows := m.matcher.MatchWindows(allWindows, identifier, identifierType)
	filteredWindows := m.matcher.FilterWindows(matchingWindows)

	return filteredWindows, nil
}

// CycleForward cycles to the next window of the same application
func (m *SimpleManager) CycleForward() (ipc.WindowInfo, error) {
	log := logger.GetDefaultLogger()
	log.LogDebug("Starting cycle forward")

	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		log.LogError("Failed to get current app identifier", "error", err)
		return ipc.WindowInfo{}, err
	}
	log.LogDebug("Current app identifier", "identifier", identifier, "type", identifierType)

	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		log.LogError("Failed to find matching windows", "error", err)
		return ipc.WindowInfo{}, err
	}
	log.LogDebug("Found matching windows", "count", len(windows))

	if len(windows) == 0 {
		log.LogError("No windows found", "identifier", identifier)
		return ipc.WindowInfo{}, fmt.Errorf("no windows found for %s", identifier)
	}

	if len(windows) == 1 {
		log.LogDebug("Only one window found, cannot cycle", "identifier", identifier)
		return windows[0], fmt.Errorf("only one %s window open (need at least 2 to cycle)", identifier)
	}

	currentIndex := -1
	for i, window := range windows {
		if window.Focused {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		log.LogError("Current window not found in matching windows")
		return ipc.WindowInfo{}, fmt.Errorf("current window not found in matching windows")
	}

	nextIndex := (currentIndex + 1) % len(windows)
	log.LogDebug("Cycling windows", "current_index", currentIndex, "next_index", nextIndex)

	if !m.config.WrapAround && nextIndex == 0 && currentIndex == len(windows)-1 {
		log.LogDebug("At last window and wrap around is disabled")
		return ipc.WindowInfo{}, fmt.Errorf("at last window and wrap around is disabled")
	}

	nextWindow := windows[nextIndex]
	log.LogInfo("Cycling to next window",
		"from_id", windows[currentIndex].ID,
		"from_name", windows[currentIndex].Name,
		"to_id", nextWindow.ID,
		"to_name", nextWindow.Name)

	if err := m.ipcClient.FocusWindow(nextWindow.ID); err != nil {
		log.LogError("Failed to focus window", "error", err, "window_id", nextWindow.ID)
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	log.LogInfo("Successfully cycled to next window", "window_id", nextWindow.ID, "window_name", nextWindow.Name)
	return nextWindow, nil
}

// CycleBackward cycles to the previous window of the same application
func (m *SimpleManager) CycleBackward() (ipc.WindowInfo, error) {
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	if len(windows) == 0 {
		return ipc.WindowInfo{}, fmt.Errorf("no windows found for %s", identifier)
	}

	if len(windows) == 1 {
		return windows[0], fmt.Errorf("only one %s window open (need at least 2 to cycle)", identifier)
	}

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

	prevIndex := (currentIndex - 1 + len(windows)) % len(windows)

	if !m.config.WrapAround && prevIndex == len(windows)-1 && currentIndex == 0 {
		return ipc.WindowInfo{}, fmt.Errorf("at first window and wrap around is disabled")
	}

	prevWindow := windows[prevIndex]

	if err := m.ipcClient.FocusWindow(prevWindow.ID); err != nil {
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	return prevWindow, nil
}

// JumpToIndex jumps to a specific window by index
func (m *SimpleManager) JumpToIndex(index int) (ipc.WindowInfo, error) {
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return ipc.WindowInfo{}, err
	}

	if index < 0 || index >= len(windows) {
		return ipc.WindowInfo{}, fmt.Errorf("index %d out of range (0-%d)", index, len(windows)-1)
	}

	targetWindow := windows[index]

	if err := m.ipcClient.FocusWindow(targetWindow.ID); err != nil {
		return ipc.WindowInfo{}, fmt.Errorf("failed to focus window: %w", err)
	}

	return targetWindow, nil
}

// GetCurrentState returns the current cycle state
func (m *SimpleManager) GetCurrentState() (*State, error) {
	identifier, identifierType, err := m.GetCurrentAppIdentifier()
	if err != nil {
		return nil, err
	}

	windows, err := m.FindMatchingWindows(identifier, identifierType)
	if err != nil {
		return nil, err
	}

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
	return nil
}

// UpdateConfig updates the cycle manager configuration
func (m *SimpleManager) UpdateConfig(config Config) error {
	m.config = config
	m.matcher = NewMatcher(config)
	return nil
}
