package stack

import (
	"fmt"
	"sync"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/cristianoliveira/sway-compat/pkg/storage"
)

// Manager handles window focus stack management
type Manager interface {
	// Push adds a window to the top of the stack
	Push(window ipc.WindowInfo)

	// Pop removes and returns the top window from the stack
	Pop() (ipc.WindowInfo, bool)

	// Peek returns the top window without removing it
	Peek() (ipc.WindowInfo, bool)

	// PeekPrevious returns the second window in the stack (previous window)
	PeekPrevious() (ipc.WindowInfo, bool)

	// Toggle swaps focus between current and previous window
	Toggle() (ipc.WindowInfo, bool)

	// List returns all windows in the stack
	List() []ipc.WindowInfo

	// Clear removes all windows from the stack
	Clear()

	// Size returns the number of windows in the stack
	Size() int

	// Save persists the stack to storage
	Save() error

	// Load restores the stack from storage
	Load() error
}

// Config holds stack manager configuration
type Config struct {
	// StackSize is the maximum number of windows to track
	StackSize int

	// DBPath is the path to the BoltDB database
	DBPath string

	// ExcludeApps is a list of app IDs to exclude from the stack
	ExcludeApps []string

	// IncludeOnly is a list of app IDs to exclusively include
	IncludeOnly []string
}

// StackManager is a concrete implementation of the Manager interface
type StackManager struct {
	windows   []ipc.WindowInfo
	maxSize   int
	storage   interface {
		storage.Storage
		storage.StackStorage
	}
	ipcClient ipc.Manager
	config    Config
	mu        sync.RWMutex
	log       logger.Logger
}

// FullStorage combines Storage and StackStorage interfaces
type FullStorage interface {
	storage.Storage
	storage.StackStorage
}

// NewStackManager creates a new stack manager
func NewStackManager(ipcClient ipc.Manager, storage FullStorage, config Config) *StackManager {
	if config.StackSize <= 0 {
		config.StackSize = 20 // default
	}

	return &StackManager{
		windows:   make([]ipc.WindowInfo, 0, config.StackSize),
		maxSize:   config.StackSize,
		storage:   storage,
		ipcClient: ipcClient,
		config:    config,
		log:       logger.GetDefaultLogger(),
	}
}

// Push adds a window to the top of the stack
func (m *StackManager) Push(window ipc.WindowInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.LogDebug("Pushing window to stack",
		"window_id", window.ID,
		"window_name", window.Name,
		"app_id", window.AppID)

	// Check if window should be excluded
	if m.shouldExclude(window) {
		m.log.LogDebug("Window excluded from stack",
			"window_id", window.ID,
			"app_id", window.AppID)
		return
	}

	// Remove existing occurrence of this window (deduplication)
	m.windows = m.removeWindow(window.ID)

	// Prepend to stack
	m.windows = append([]ipc.WindowInfo{window}, m.windows...)

	// Trim to max size
	if len(m.windows) > m.maxSize {
		m.windows = m.windows[:m.maxSize]
	}

	// Save to storage (open/close quickly to allow other processes to access)
	if err := m.persistPush(window.ID); err != nil {
		m.log.LogError("Failed to save window to storage",
			"error", err,
			"window_id", window.ID)
	}

	m.log.LogInfo("Window pushed to stack",
		"window_id", window.ID,
		"stack_size", len(m.windows))
}

// Pop removes and returns the top window from the stack
func (m *StackManager) Pop() (ipc.WindowInfo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.windows) == 0 {
		return ipc.WindowInfo{}, false
	}

	window := m.windows[0]
	m.windows = m.windows[1:]

	return window, true
}

// Peek returns the top window without removing it
func (m *StackManager) Peek() (ipc.WindowInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.windows) == 0 {
		return ipc.WindowInfo{}, false
	}

	return m.windows[0], true
}

// PeekPrevious returns the second window in the stack (previous window)
func (m *StackManager) PeekPrevious() (ipc.WindowInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.windows) < 2 {
		return ipc.WindowInfo{}, false
	}

	return m.windows[1], true
}

// Toggle swaps focus between current and previous window
func (m *StackManager) Toggle() (ipc.WindowInfo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.LogDebug("Toggling to previous window")

	if len(m.windows) < 2 {
		m.log.LogDebug("Not enough windows in stack to toggle")
		return ipc.WindowInfo{}, false
	}

	// Get the previous window (index 1)
	previousWindow := m.windows[1]

	// Validate window still exists
	validWindow, exists := m.validateWindow(previousWindow.ID)
	if !exists {
		m.log.LogDebug("Previous window no longer exists, removing from stack",
			"window_id", previousWindow.ID)

		// Remove invalid window and try next
		m.windows = append(m.windows[:1], m.windows[2:]...)

		// Try again with the new "previous" window
		if len(m.windows) < 2 {
			return ipc.WindowInfo{}, false
		}

		previousWindow = m.windows[1]
		validWindow, exists = m.validateWindow(previousWindow.ID)
		if !exists {
			m.log.LogError("Multiple invalid windows in stack")
			return ipc.WindowInfo{}, false
		}
	}

	// Focus the previous window
	if err := m.ipcClient.FocusWindow(previousWindow.ID); err != nil {
		m.log.LogError("Failed to focus window",
			"error", err,
			"window_id", previousWindow.ID)
		return ipc.WindowInfo{}, false
	}

	m.log.LogInfo("Toggled to previous window",
		"window_id", previousWindow.ID,
		"window_name", previousWindow.Name)

	return *validWindow, true
}

// List returns all windows in the stack
func (m *StackManager) List() []ipc.WindowInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make([]ipc.WindowInfo, len(m.windows))
	copy(result, m.windows)
	return result
}

// Clear removes all windows from the stack
func (m *StackManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.windows = make([]ipc.WindowInfo, 0, m.maxSize)

	if err := m.storage.ClearStack(); err != nil {
		m.log.LogError("Failed to clear storage", "error", err)
	}

	m.log.LogInfo("Stack cleared")
}

// Size returns the number of windows in the stack
func (m *StackManager) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.windows)
}

// Save persists the stack to storage
func (m *StackManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// FileStorage handles its own locking
	// Clear storage first
	if err := m.storage.ClearStack(); err != nil {
		return fmt.Errorf("failed to clear storage: %w", err)
	}

	// Push windows in reverse order (so they end up in correct order)
	for i := len(m.windows) - 1; i >= 0; i-- {
		if err := m.storage.PushStack(m.windows[i].ID); err != nil {
			return fmt.Errorf("failed to save window %d: %w", m.windows[i].ID, err)
		}
	}

	m.log.LogDebug("Stack saved to storage", "size", len(m.windows))
	return nil
}

// Load restores the stack from storage
func (m *StackManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.LogDebug("Loading stack from storage")

	// Get window IDs from storage (FileStorage handles its own locking)
	ids, err := m.storage.GetStack()
	if err != nil {
		return fmt.Errorf("failed to get stack from storage: %w", err)
	}

	if len(ids) == 0 {
		m.log.LogDebug("No windows in storage")
		return nil
	}

	// Get window tree to validate and get window info
	tree, err := m.ipcClient.GetTree()
	if err != nil {
		return fmt.Errorf("failed to get window tree: %w", err)
	}

	allWindows := ipc.FindAllWindows(tree)

	// Build windows slice from IDs
	m.windows = make([]ipc.WindowInfo, 0, len(ids))
	for _, id := range ids {
		// Find window info
		var found *ipc.WindowInfo
		for i := range allWindows {
			if allWindows[i].ID == id {
				found = &allWindows[i]
				break
			}
		}

		if found != nil {
			m.windows = append(m.windows, *found)
		} else {
			m.log.LogDebug("Window from storage no longer exists", "window_id", id)
		}
	}

	m.log.LogInfo("Stack loaded from storage",
		"ids_count", len(ids),
		"windows_count", len(m.windows))

	return nil
}

// shouldExclude checks if a window should be excluded from the stack
func (m *StackManager) shouldExclude(window ipc.WindowInfo) bool {
	// Check ExcludeApps
	for _, appID := range m.config.ExcludeApps {
		if window.AppID == appID {
			return true
		}
	}

	// Check IncludeOnly (if specified, only these are allowed)
	if len(m.config.IncludeOnly) > 0 {
		found := false
		for _, appID := range m.config.IncludeOnly {
			if window.AppID == appID {
				found = true
				break
			}
		}
		return !found
	}

	return false
}

// removeWindow removes a window by ID from the stack
func (m *StackManager) removeWindow(windowID int64) []ipc.WindowInfo {
	result := make([]ipc.WindowInfo, 0, len(m.windows))
	for _, w := range m.windows {
		if w.ID != windowID {
			result = append(result, w)
		}
	}
	return result
}

// validateWindow checks if a window still exists in the window tree
func (m *StackManager) validateWindow(windowID int64) (*ipc.WindowInfo, bool) {
	tree, err := m.ipcClient.GetTree()
	if err != nil {
		m.log.LogError("Failed to get window tree for validation", "error", err)
		return nil, false
	}

	windows := ipc.FindAllWindows(tree)
	for i := range windows {
		if windows[i].ID == windowID {
			return &windows[i], true
		}
	}

	return nil, false
}

// persistPush saves window ID to storage (FileStorage handles locking internally)
func (m *StackManager) persistPush(windowID int64) error {
	// FileStorage handles its own file locking on each operation
	return m.storage.PushStack(windowID)
}
