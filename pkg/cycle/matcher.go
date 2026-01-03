package cycle

import (
	"regexp"

	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

// Matcher handles window matching logic
type Matcher struct {
	config Config
}

// NewMatcher creates a new window matcher
func NewMatcher(config Config) *Matcher {
	// Set defaults if not provided
	if len(config.IdentifierPriority) == 0 {
		config.IdentifierPriority = []string{"app_id", "class", "instance"}
	}
	return &Matcher{config: config}
}

// ExtractIdentifier extracts the identifier from a window based on priority
func (m *Matcher) ExtractIdentifier(window ipc.WindowInfo) (string, string) {
	for _, identifierType := range m.config.IdentifierPriority {
		switch identifierType {
		case "app_id":
			if window.AppID != "" {
				return window.AppID, "app_id"
			}
		case "class":
			if window.Class != "" {
				return window.Class, "class"
			}
		case "instance":
			if window.Instance != "" {
				return window.Instance, "instance"
			}
		}
	}
	// Fallback to name if no identifier found
	return window.Name, "name"
}

// MatchWindows finds all windows matching the given identifier
func (m *Matcher) MatchWindows(windows []ipc.WindowInfo, identifier, identifierType string) []ipc.WindowInfo {
	var matches []ipc.WindowInfo

	for _, window := range windows {
		// Get the identifier value based on type
		var windowIdentifier string
		switch identifierType {
		case "app_id":
			windowIdentifier = window.AppID
		case "class":
			windowIdentifier = window.Class
		case "instance":
			windowIdentifier = window.Instance
		case "name":
			windowIdentifier = window.Name
		}

		// Check if it matches
		if windowIdentifier == identifier {
			matches = append(matches, window)
		}
	}

	return matches
}

// FilterWindows applies filters to the window list
func (m *Matcher) FilterWindows(windows []ipc.WindowInfo) []ipc.WindowInfo {
	var filtered []ipc.WindowInfo

	for _, window := range windows {
		// Apply exclude scratchpad filter
		if m.config.ExcludeScratchpad && window.Scratchpad != "" && window.Scratchpad != "none" {
			continue
		}

		// Apply exclude minimized filter (windows with no visible flag)
		if m.config.ExcludeMinimized && !window.Visible {
			continue
		}

		// Apply app-specific rules if any
		if rule, ok := m.config.AppRules[window.AppID]; ok {
			// Check exclude regex
			if rule.ExcludeRegex != "" {
				matched, err := regexp.MatchString(rule.ExcludeRegex, window.Name)
				if err == nil && matched {
					continue
				}
			}
		}

		filtered = append(filtered, window)
	}

	return filtered
}

// GetIdentifierForApp returns the identifier to use for a specific app
func (m *Matcher) GetIdentifierForApp(appID string) string {
	if rule, ok := m.config.AppRules[appID]; ok {
		if rule.MatchBy != "" {
			return rule.MatchBy
		}
	}
	// Return default priority
	if len(m.config.IdentifierPriority) > 0 {
		return m.config.IdentifierPriority[0]
	}
	return "app_id"
}
