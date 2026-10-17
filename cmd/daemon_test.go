package cmd

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

func init() {
	logger.SetDefaultLogger(&logger.EmptyLogger{})
}

type mockIPCManager struct {
	focusCalls   []int64
	commandCalls []string
	commandError error
	marks        []string
	marksError   error
	focused      *ipc.WindowInfo
}

func newMockIPCManager() *mockIPCManager {
	return &mockIPCManager{
		focusCalls:   []int64{},
		commandCalls: []string{},
	}
}

func (m *mockIPCManager) Connect() error                             { return nil }
func (m *mockIPCManager) GetTree() (*ipc.Tree, error)                { return nil, nil }
func (m *mockIPCManager) GetFocusedWindow() (*ipc.WindowInfo, error) { return m.focused, nil }
func (m *mockIPCManager) Subscribe(events []string) (chan ipc.Event, error) {
	return nil, nil
}
func (m *mockIPCManager) GetWindowInfo(id int64) (*ipc.WindowInfo, error) { return nil, nil }
func (m *mockIPCManager) GetMarks() ([]string, error)                     { return m.marks, m.marksError }
func (m *mockIPCManager) Close() error                                    { return nil }

func (m *mockIPCManager) FocusWindow(id int64) error {
	m.focusCalls = append(m.focusCalls, id)
	return nil
}

func (m *mockIPCManager) RunCommand(cmd string) error {
	m.commandCalls = append(m.commandCalls, cmd)
	return m.commandError
}

func TestHandleFloatingFocusHidesPreviousFloatingWindow(t *testing.T) {
	client := newMockIPCManager()
	prev := &ipc.WindowInfo{ID: 10, Floating: true, Scratchpad: "none", Workspace: "2"}
	curr := &ipc.WindowInfo{ID: 20, Workspace: "2"}

	handleFloatingFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.commandCalls) != 1 || client.commandCalls[0] != "[con_id=10] move scratchpad" {
		t.Fatalf("expected previous floating window to move to scratchpad, got %v", client.commandCalls)
	}
	if len(client.focusCalls) != 0 {
		t.Fatalf("expected no re-focus, got %v", client.focusCalls)
	}
}

func TestHandleFloatingFocusIgnoresAnotherWindowPin(t *testing.T) {
	client := newMockIPCManager()
	client.marks = []string{"_sway_compat_pinned_99", "other-mark"}
	prev := &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}
	curr := &ipc.WindowInfo{ID: 20, Workspace: "2"}

	handleFloatingFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.commandCalls) != 1 || client.commandCalls[0] != "[con_id=10] move scratchpad" {
		t.Fatalf("expected unpinned window to hide, got %v", client.commandCalls)
	}
}

func TestHandleFloatingFocusKeepsPinnedWindowVisible(t *testing.T) {
	client := newMockIPCManager()
	client.marks = []string{"_sway_compat_pinned_10"}
	prev := &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}
	curr := &ipc.WindowInfo{ID: 20, Workspace: "2"}

	handleFloatingFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.commandCalls) != 0 {
		t.Fatalf("expected pinned window to stay visible, got %v", client.commandCalls)
	}
}

func TestHandleFloatingFocusKeepsWindowVisibleWhenMarksUnavailable(t *testing.T) {
	client := newMockIPCManager()
	client.marksError = errors.New("Sway IPC unavailable")
	prev := &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}
	curr := &ipc.WindowInfo{ID: 20, Workspace: "2"}

	handleFloatingFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.commandCalls) != 0 {
		t.Fatalf("expected no hide when pin state is unknown, got %v", client.commandCalls)
	}
}

func TestHandleFloatingFocusSkipsOtherTransitions(t *testing.T) {
	tests := []struct {
		name string
		prev *ipc.WindowInfo
		curr *ipc.WindowInfo
	}{
		{"no previous window", nil, &ipc.WindowInfo{ID: 20, Workspace: "2"}},
		{"previous tiled", &ipc.WindowInfo{ID: 10, Workspace: "2"}, &ipc.WindowInfo{ID: 20, Workspace: "2"}},
		{"current floating", &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}, &ipc.WindowInfo{ID: 20, Floating: true, Workspace: "2"}},
		{"different workspace", &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "1"}, &ipc.WindowInfo{ID: 20, Workspace: "2"}},
		{"missing previous workspace", &ipc.WindowInfo{ID: 10, Floating: true}, &ipc.WindowInfo{ID: 20, Workspace: "2"}},
		{"missing current workspace", &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}, &ipc.WindowInfo{ID: 20}},
		{"same window", &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}, &ipc.WindowInfo{ID: 10, Workspace: "2"}},
		{"no previous window id", &ipc.WindowInfo{Floating: true, Workspace: "2"}, &ipc.WindowInfo{ID: 20, Workspace: "2"}},
		{"no current window id", &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}, &ipc.WindowInfo{Workspace: "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockIPCManager()
			handleFloatingFocus(tt.prev, tt.curr, client, logger.GetDefaultLogger())
			if len(client.commandCalls) != 0 {
				t.Fatalf("expected no command, got %v", client.commandCalls)
			}
		})
	}
}

func TestHandleFloatingFocusDoesNotRefocusWhenCommandFails(t *testing.T) {
	client := newMockIPCManager()
	client.commandError = errors.New("sway command failed")
	prev := &ipc.WindowInfo{ID: 10, Floating: true, Workspace: "2"}
	curr := &ipc.WindowInfo{ID: 20, Workspace: "2"}

	handleFloatingFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.commandCalls) != 1 || len(client.focusCalls) != 0 {
		t.Fatalf("expected one attempted command and no focus change, got commands %v, focus %v", client.commandCalls, client.focusCalls)
	}
}

func TestHandleScratchpadFocusSkipsWhenWorkspaceDiffers(t *testing.T) {
	client := newMockIPCManager()
	prev := &ipc.WindowInfo{
		ID:         10,
		Floating:   true,
		Scratchpad: "shown",
		Workspace:  "1",
	}
	curr := &ipc.WindowInfo{
		ID:        20,
		Floating:  false,
		Workspace: "2",
	}

	daemonScratchpadHideAction = "hide-scratchpad"
	handleScratchpadFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.focusCalls) != 0 {
		t.Fatalf("expected no focus calls, got %d", len(client.focusCalls))
	}
	if len(client.commandCalls) != 0 {
		t.Fatalf("expected no scratchpad commands, got %d", len(client.commandCalls))
	}
}

func TestHandleScratchpadFocusSkipsWhenWorkspaceUnknown(t *testing.T) {
	client := newMockIPCManager()
	prev := &ipc.WindowInfo{
		ID:         10,
		Floating:   true,
		Scratchpad: "shown",
		Workspace:  "",
	}
	curr := &ipc.WindowInfo{
		ID:        20,
		Floating:  false,
		Workspace: "",
	}

	daemonScratchpadHideAction = "hide-scratchpad"
	handleScratchpadFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.focusCalls) != 0 {
		t.Fatalf("expected no focus calls, got %d", len(client.focusCalls))
	}
	if len(client.commandCalls) != 0 {
		t.Fatalf("expected no scratchpad commands, got %d", len(client.commandCalls))
	}
}

func TestHandleScratchpadFocusExecutesSameWorkspace(t *testing.T) {
	client := newMockIPCManager()
	prev := &ipc.WindowInfo{
		ID:         10,
		Floating:   true,
		Scratchpad: "shown",
		Workspace:  "2",
	}
	curr := &ipc.WindowInfo{
		ID:        20,
		Floating:  false,
		Workspace: "2",
	}

	daemonScratchpadHideAction = "hide-scratchpad"
	handleScratchpadFocus(prev, curr, client, logger.GetDefaultLogger())

	if len(client.focusCalls) != 1 {
		t.Fatalf("expected one focus call, got %d", len(client.focusCalls))
	}
	if client.focusCalls[0] != curr.ID {
		t.Fatalf("expected focus on %d, got %d", curr.ID, client.focusCalls[0])
	}

	if len(client.commandCalls) != 1 {
		t.Fatalf("expected one scratchpad command, got %d", len(client.commandCalls))
	}

	expectedCmd := "[con_id=10] move scratchpad"
	if client.commandCalls[0] != expectedCmd {
		t.Fatalf("expected command %q, got %q", expectedCmd, client.commandCalls[0])
	}
}
