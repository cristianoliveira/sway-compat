package stack

import "github.com/cristianoliveira/sway-compat/pkg/ipc"

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
