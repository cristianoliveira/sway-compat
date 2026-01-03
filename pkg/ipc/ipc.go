package ipc

// Manager handles communication with Sway IPC
type Manager interface {
	// Connect establishes connection to Sway IPC socket
	Connect() error

	// GetTree retrieves the window tree from Sway
	GetTree() (*Tree, error)

	// GetFocusedWindow returns the currently focused window
	GetFocusedWindow() (*WindowInfo, error)

	// FocusWindow focuses the window with the given ID
	FocusWindow(id int64) error

	// Subscribe subscribes to Sway events
	Subscribe(events []string) (chan Event, error)

	// Close closes the IPC connection
	Close() error
}

// WindowInfo represents a window in the Sway tree
type WindowInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AppID      string `json:"app_id"`
	Class      string `json:"class"`
	Instance   string `json:"instance"`
	Type       string `json:"type"`
	Workspace  string `json:"workspace"`
	Focused    bool   `json:"focused"`
	Visible    bool   `json:"visible"`
	Scratchpad string `json:"scratchpad_state"`
}

// Tree represents the Sway window tree
type Tree struct {
	ID       int64        `json:"id"`
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	Focused  bool         `json:"focused"`
	Nodes    []*Tree      `json:"nodes"`
	AppID    string       `json:"app_id"`
	Class    string       `json:"class"`
	Instance string       `json:"instance"`
	Window   int64        `json:"window"`
}

// Event represents a Sway IPC event
type Event struct {
	Change    string      `json:"change"`
	Container *WindowInfo `json:"container"`
}
