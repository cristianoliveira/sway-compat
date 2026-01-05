package cmd

import (
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
}

func newMockIPCManager() *mockIPCManager {
	return &mockIPCManager{
		focusCalls:   []int64{},
		commandCalls: []string{},
	}
}

func (m *mockIPCManager) Connect() error                             { return nil }
func (m *mockIPCManager) GetTree() (*ipc.Tree, error)                { return nil, nil }
func (m *mockIPCManager) GetFocusedWindow() (*ipc.WindowInfo, error) { return nil, nil }
func (m *mockIPCManager) Subscribe(events []string) (chan ipc.Event, error) {
	return nil, nil
}
func (m *mockIPCManager) GetWindowInfo(id int64) (*ipc.WindowInfo, error) { return nil, nil }
func (m *mockIPCManager) Close() error                                    { return nil }

func (m *mockIPCManager) FocusWindow(id int64) error {
	m.focusCalls = append(m.focusCalls, id)
	return nil
}

func (m *mockIPCManager) RunCommand(cmd string) error {
	m.commandCalls = append(m.commandCalls, cmd)
	return nil
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
