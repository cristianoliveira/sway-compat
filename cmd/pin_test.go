package cmd

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

func TestSetFocusedWindowPinnedPreservesOtherMarks(t *testing.T) {
	client := newMockIPCManager()
	client.focused = &ipc.WindowInfo{ID: 42}

	id, err := setFocusedWindowPinned(client, true)
	if err != nil || id != 42 {
		t.Fatalf("expected to pin window 42, got id %d, error %v", id, err)
	}
	if len(client.commandCalls) != 1 || client.commandCalls[0] != "[con_id=42] mark --add _sway_compat_pinned_42" {
		t.Fatalf("expected additive mark for window 42, got %v", client.commandCalls)
	}
}

func TestSetFocusedWindowUnpinnedRemovesOnlyItsOwnMark(t *testing.T) {
	client := newMockIPCManager()
	client.focused = &ipc.WindowInfo{ID: 42}

	_, err := setFocusedWindowPinned(client, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(client.commandCalls) != 1 || client.commandCalls[0] != "[con_id=42] unmark _sway_compat_pinned_42" {
		t.Fatalf("expected targeted unmark, got %v", client.commandCalls)
	}
}

func TestSetFocusedWindowPinnedRejectsMissingWindow(t *testing.T) {
	for _, focused := range []*ipc.WindowInfo{nil, {ID: 0}} {
		client := newMockIPCManager()
		client.focused = focused

		if _, err := setFocusedWindowPinned(client, true); err == nil {
			t.Fatal("expected an error without a focused window ID")
		}
		if len(client.commandCalls) != 0 {
			t.Fatalf("expected no command, got %v", client.commandCalls)
		}
	}
}

func TestSetFocusedWindowPinnedReportsCommandFailure(t *testing.T) {
	client := newMockIPCManager()
	client.focused = &ipc.WindowInfo{ID: 42}
	client.commandError = errors.New("Sway command failed")

	if _, err := setFocusedWindowPinned(client, true); err == nil {
		t.Fatal("expected command failure")
	}
}
