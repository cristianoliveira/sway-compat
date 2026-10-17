package cmd

import (
	"fmt"

	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/spf13/cobra"
)

func pinnedWindowMark(id int64) string {
	return fmt.Sprintf("_sway_compat_pinned_%d", id)
}

func setFocusedWindowPinned(client ipc.Manager, pin bool) (int64, error) {
	window, err := client.GetFocusedWindow()
	if err != nil {
		return 0, fmt.Errorf("get focused window: %w", err)
	}
	if window == nil || window.ID <= 0 {
		return 0, fmt.Errorf("no focused window to pin or unpin")
	}

	command := fmt.Sprintf("[con_id=%d] mark --add %s", window.ID, pinnedWindowMark(window.ID))
	if !pin {
		command = fmt.Sprintf("[con_id=%d] unmark %s", window.ID, pinnedWindowMark(window.ID))
	}
	if err := client.RunCommand(command); err != nil {
		return 0, fmt.Errorf("change window pin: %w", err)
	}
	return window.ID, nil
}

func runPin(pin bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		client, err := ipc.NewClient()
		if err != nil {
			return fmt.Errorf("connect to Sway: %w", err)
		}
		defer client.Close()

		id, err := setFocusedWindowPinned(client, pin)
		if err != nil {
			return err
		}
		if pin {
			fmt.Fprintf(cmd.OutOrStdout(), "Pinned window %d\n", id)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Unpinned window %d\n", id)
		}
		return nil
	}
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "pin",
		Short: "Keep the focused window visible when a tiled window takes focus",
		Long:  "Pin the focused window so daemon --hide-floating-on-focus does not hide it. The pin lasts until you unpin the window or close it.",
		Args:  cobra.NoArgs,
		RunE:  runPin(true),
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "unpin",
		Short: "Allow the focused floating window to hide on tiled focus",
		Long:  "Unpin the focused window so daemon --hide-floating-on-focus can move it to Sway's scratchpad when a tiled window takes focus.",
		Args:  cobra.NoArgs,
		RunE:  runPin(false),
	})
}
