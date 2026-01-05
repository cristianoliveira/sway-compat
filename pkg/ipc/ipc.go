package ipc

import "encoding/json"

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

	// RunCommand runs a Sway command
	RunCommand(cmd string) error

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
	Floating   bool   `json:"floating"`
}

// UnmarshalJSON implements custom JSON unmarshaling for WindowInfo
// to handle Sway's floating field which can be a string ("user_on", "user_off", etc.)
func (w *WindowInfo) UnmarshalJSON(data []byte) error {
	type Alias WindowInfo
	aux := &struct {
		Floating interface{} `json:"floating"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle floating field which can be bool or string
	switch v := aux.Floating.(type) {
	case bool:
		w.Floating = v
	case string:
		// "user_on", "auto_on" indicate floating
		w.Floating = v == "user_on" || v == "auto_on"
	case nil:
		w.Floating = false
	default:
		w.Floating = false
	}

	return nil
}

// Tree represents the Sway window tree
type Tree struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Focused    bool    `json:"focused"`
	Nodes      []*Tree `json:"nodes"`
	AppID      string  `json:"app_id"`
	Class      string  `json:"class"`
	Instance   string  `json:"instance"`
	Window     int64   `json:"window"`
	Workspace  string  `json:"workspace"`
	Scratchpad string  `json:"scratchpad_state"`
	Floating   bool    `json:"floating"`
}

// Event represents a Sway IPC event
type Event struct {
	Change    string      `json:"change"`
	Container *WindowInfo `json:"container"`
}
