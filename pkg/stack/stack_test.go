package stack

import "testing"

func TestConfig(t *testing.T) {
	config := Config{
		StackSize: 20,
		DBPath:    "/tmp/test.db",
	}

	if config.StackSize != 20 {
		t.Errorf("Expected stack size 20, got %d", config.StackSize)
	}

	if config.DBPath != "/tmp/test.db" {
		t.Errorf("Expected DB path '/tmp/test.db', got '%s'", config.DBPath)
	}
}
