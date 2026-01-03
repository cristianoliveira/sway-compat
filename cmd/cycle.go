package cmd

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/sway-compat/pkg/cycle"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/spf13/cobra"
)

var (
	quietFlag bool
)

var cycleCmd = &cobra.Command{
	Use:   "cycle",
	Short: "Cycle through windows of the same application",
	Long: `Cycle through windows of the same application, similar to macOS Cmd+` + "`" + ` behavior.
This command finds all windows with the same app_id or class as the currently
focused window and cycles to the next one.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCycleForward()
	},
}

var cycleForwardCmd = &cobra.Command{
	Use:   "cycle-forward",
	Short: "Cycle forward to next window of same app",
	Run: func(cmd *cobra.Command, args []string) {
		runCycleForward()
	},
}

var cycleBackwardCmd = &cobra.Command{
	Use:   "cycle-backward",
	Short: "Cycle backward to previous window of same app",
	Run: func(cmd *cobra.Command, args []string) {
		runCycleBackward()
	},
}

func runCycleForward() {
	// Create IPC client
	client, err := ipc.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Create cycle manager with default config
	config := cycle.Config{
		IdentifierPriority: []string{"app_id", "class", "instance"},
		ExcludeScratchpad:  true,
		ExcludeMinimized:   false,
		WrapAround:         true,
	}
	manager := cycle.NewSimpleManager(client, config)

	// Cycle forward
	window, err := manager.CycleForward()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Focused window: %s (ID: %d)\n", window.Name, window.ID)
}

func runCycleBackward() {
	// Create IPC client
	client, err := ipc.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Create cycle manager with default config
	config := cycle.Config{
		IdentifierPriority: []string{"app_id", "class", "instance"},
		ExcludeScratchpad:  true,
		ExcludeMinimized:   false,
		WrapAround:         true,
	}
	manager := cycle.NewSimpleManager(client, config)

	// Cycle backward
	window, err := manager.CycleBackward()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Focused window: %s (ID: %d)\n", window.Name, window.ID)
}

func init() {
	rootCmd.AddCommand(cycleCmd)
	rootCmd.AddCommand(cycleForwardCmd)
	rootCmd.AddCommand(cycleBackwardCmd)

	// Add flags for cycle commands if needed
}
