package ipc

import "testing"

func TestWindowInfo(t *testing.T) {
	window := WindowInfo{
		ID:    123,
		Name:  "Test Window",
		AppID: "test-app",
	}

	if window.ID != 123 {
		t.Errorf("Expected window ID 123, got %d", window.ID)
	}

	if window.AppID != "test-app" {
		t.Errorf("Expected app ID 'test-app', got '%s'", window.AppID)
	}
}
