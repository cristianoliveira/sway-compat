package cycle

import "github.com/cristianoliveira/sway-compat/pkg/ipc"

// Manager handles window cycling within the same application
type Manager interface {
	// GetCurrentAppIdentifier returns the identifier of the currently focused app
	GetCurrentAppIdentifier() (string, string, error)

	// FindMatchingWindows finds all windows matching the given identifier
	FindMatchingWindows(identifier, identifierType string) ([]ipc.WindowInfo, error)

	// CycleForward cycles to the next window of the same application
	CycleForward() (ipc.WindowInfo, error)

	// CycleBackward cycles to the previous window of the same application
	CycleBackward() (ipc.WindowInfo, error)

	// JumpToIndex jumps to a specific window by index
	JumpToIndex(index int) (ipc.WindowInfo, error)

	// GetCurrentState returns the current cycle state
	GetCurrentState() (*State, error)

	// ClearState clears the current cycle state
	ClearState() error

	// UpdateConfig updates the cycle manager configuration
	UpdateConfig(config Config) error
}

// State represents the current cycling state
type State struct {
	CurrentAppID string            `json:"current_app_id"`
	Windows      []ipc.WindowInfo  `json:"windows"`
	CurrentIndex int               `json:"current_index"`
}

// Config holds cycle manager configuration
type Config struct {
	// DBPath is the path to the BoltDB database
	DBPath string

	// IdentifierPriority defines the order of preference for window identifiers
	IdentifierPriority []string

	// ExcludeScratchpad excludes scratchpad windows from cycling
	ExcludeScratchpad bool

	// ExcludeMinimized excludes minimized windows from cycling
	ExcludeMinimized bool

	// WrapAround enables wrapping around to the first window after the last
	WrapAround bool

	// MaintainOrder maintains MRU (most recently used) order
	MaintainOrder bool

	// AppRules defines application-specific cycling rules
	AppRules map[string]AppRule
}

// AppRule defines rules for a specific application
type AppRule struct {
	// MatchBy overrides the identifier priority for this app
	MatchBy string

	// ExcludeRegex excludes windows matching this regex pattern
	ExcludeRegex string

	// GroupBy defines how to group windows ("workspace", "monitor", "none")
	GroupBy string
}
