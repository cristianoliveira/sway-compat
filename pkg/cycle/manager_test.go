package cycle

import (
	"testing"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

func init() {
	// Initialize empty logger for tests
	logger.SetDefaultLogger(&logger.EmptyLogger{})
}

// MockIPCManager is a mock implementation for testing
type MockIPCManager struct {
	tree          *ipc.Tree
	focusedWindow *ipc.WindowInfo
	lastFocusedID int64
}

func (m *MockIPCManager) Connect() error {
	return nil
}

func (m *MockIPCManager) GetTree() (*ipc.Tree, error) {
	return m.tree, nil
}

func (m *MockIPCManager) GetFocusedWindow() (*ipc.WindowInfo, error) {
	return m.focusedWindow, nil
}

func (m *MockIPCManager) FocusWindow(id int64) error {
	m.lastFocusedID = id
	return nil
}

func (m *MockIPCManager) Subscribe(events []string) (chan ipc.Event, error) {
	return make(chan ipc.Event), nil
}

func (m *MockIPCManager) Close() error {
	return nil
}

func TestSimpleManager_GetCurrentAppIdentifier(t *testing.T) {
	mockIPC := &MockIPCManager{
		focusedWindow: &ipc.WindowInfo{
			ID:    1,
			Name:  "Test Window",
			AppID: "test-app",
		},
	}

	config := Config{
		IdentifierPriority: []string{"app_id", "class", "instance"},
	}

	manager := NewSimpleManager(mockIPC, config)

	identifier, identifierType, err := manager.GetCurrentAppIdentifier()
	if err != nil {
		t.Fatalf("Failed to get identifier: %v", err)
	}

	if identifier != "test-app" {
		t.Errorf("Expected identifier 'test-app', got '%s'", identifier)
	}

	if identifierType != "app_id" {
		t.Errorf("Expected type 'app_id', got '%s'", identifierType)
	}
}

func TestSimpleManager_FindMatchingWindows(t *testing.T) {
	// Create a mock tree with multiple windows
	mockTree := &ipc.Tree{
		ID:   0,
		Type: "root",
		Nodes: []*ipc.Tree{
			{
				ID:      1,
				Type:    "con",
				AppID:   "firefox",
				Name:    "Firefox 1",
				Window:  101,
				Focused: true,
			},
			{
				ID:     2,
				Type:   "con",
				AppID:  "firefox",
				Name:   "Firefox 2",
				Window: 102,
			},
			{
				ID:     3,
				Type:   "con",
				AppID:  "terminal",
				Name:   "Terminal",
				Window: 103,
			},
		},
	}

	mockIPC := &MockIPCManager{
		tree: mockTree,
	}

	config := Config{
		IdentifierPriority: []string{"app_id"},
		ExcludeScratchpad:  true,
		WrapAround:         true,
	}

	manager := NewSimpleManager(mockIPC, config)

	windows, err := manager.FindMatchingWindows("firefox", "app_id")
	if err != nil {
		t.Fatalf("Failed to find windows: %v", err)
	}

	if len(windows) != 2 {
		t.Errorf("Expected 2 firefox windows, got %d", len(windows))
	}
}

func TestMatcher_ExtractIdentifier(t *testing.T) {
	config := Config{
		IdentifierPriority: []string{"app_id", "class", "instance"},
	}

	matcher := NewMatcher(config)

	tests := []struct {
		name             string
		window           ipc.WindowInfo
		expectedID       string
		expectedType     string
	}{
		{
			name: "app_id priority",
			window: ipc.WindowInfo{
				AppID:    "firefox",
				Class:    "Firefox",
				Instance: "firefox-1",
			},
			expectedID:   "firefox",
			expectedType: "app_id",
		},
		{
			name: "class fallback",
			window: ipc.WindowInfo{
				AppID:    "",
				Class:    "Firefox",
				Instance: "firefox-1",
			},
			expectedID:   "Firefox",
			expectedType: "class",
		},
		{
			name: "instance fallback",
			window: ipc.WindowInfo{
				AppID:    "",
				Class:    "",
				Instance: "firefox-1",
			},
			expectedID:   "firefox-1",
			expectedType: "instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, idType := matcher.ExtractIdentifier(tt.window)
			if id != tt.expectedID {
				t.Errorf("Expected id '%s', got '%s'", tt.expectedID, id)
			}
			if idType != tt.expectedType {
				t.Errorf("Expected type '%s', got '%s'", tt.expectedType, idType)
			}
		})
	}
}
