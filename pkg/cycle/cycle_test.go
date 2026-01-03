package cycle

import "testing"

func TestConfig(t *testing.T) {
	config := Config{
		DBPath:             "/tmp/test.db",
		ExcludeScratchpad:  true,
		ExcludeMinimized:   true,
		WrapAround:         true,
		IdentifierPriority: []string{"app_id", "class", "instance"},
	}

	if config.DBPath != "/tmp/test.db" {
		t.Errorf("Expected DB path '/tmp/test.db', got '%s'", config.DBPath)
	}

	if !config.ExcludeScratchpad {
		t.Error("Expected ExcludeScratchpad to be true")
	}

	if len(config.IdentifierPriority) != 3 {
		t.Errorf("Expected 3 identifier priorities, got %d", len(config.IdentifierPriority))
	}
}

func TestAppRule(t *testing.T) {
	rule := AppRule{
		MatchBy:      "app_id",
		ExcludeRegex: "^terminal-scratchpad$",
		GroupBy:      "workspace",
	}

	if rule.MatchBy != "app_id" {
		t.Errorf("Expected MatchBy 'app_id', got '%s'", rule.MatchBy)
	}

	if rule.GroupBy != "workspace" {
		t.Errorf("Expected GroupBy 'workspace', got '%s'", rule.GroupBy)
	}
}
