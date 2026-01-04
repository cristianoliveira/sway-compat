package ipc

// Request represents an IPC request from a command to the daemon
type Request struct {
	Type     string `json:"type"`                // TOGGLE, LIST, CLEAR, PUSH
	WindowID *int64 `json:"window_id,omitempty"` // For PUSH requests
}

// Response represents an IPC response from the daemon to a command
type Response struct {
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
	Window  *WindowInfo  `json:"window,omitempty"`  // For TOGGLE
	Windows []WindowInfo `json:"windows,omitempty"` // For LIST
}

// Request types
const (
	RequestTypeToggle = "TOGGLE"
	RequestTypeList   = "LIST"
	RequestTypeClear  = "CLEAR"
	RequestTypePush   = "PUSH"
)
