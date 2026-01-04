package cmd

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage window focus stack",
	Long: `Window stack management for Alt+Tab style switching.
Maintains a history of focused windows and allows toggling between them.`,
}

var stackToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle between current and previous window",
	Long: `Switch focus between the current window and the previously focused window.
This is typically bound to Mod+Tab for quick window switching.`,
	Run: runStackToggle,
}

func runStackToggle(cmd *cobra.Command, args []string) {
	log := logger.GetDefaultLogger()
	log.LogDebug("Stack toggle requested")

	// Connect to daemon
	client := ipc.NewDaemonClient()
	socketPath := getSocketPath()

	if err := client.Connect(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure the daemon is running: sway-compat daemon\n")
		os.Exit(1)
	}
	defer client.Close()

	// Send toggle request
	window, err := client.Toggle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.LogInfo("Toggled to window", "window_id", window.ID, "window_name", window.Name)
	fmt.Printf("Switched to: %s (ID: %d)\n", window.Name, window.ID)
}

var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current window stack",
	Run: runStackList,
}

func runStackList(cmd *cobra.Command, args []string) {
	log := logger.GetDefaultLogger()
	log.LogDebug("Stack list requested")

	// Connect to daemon
	client := ipc.NewDaemonClient()
	socketPath := getSocketPath()

	if err := client.Connect(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure the daemon is running: sway-compat daemon\n")
		os.Exit(1)
	}
	defer client.Close()

	// Send list request
	windows, err := client.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(windows) == 0 {
		fmt.Println("Window stack is empty")
		return
	}

	fmt.Printf("Window Stack (%d windows):\n", len(windows))
	for i, window := range windows {
		marker := " "
		if i == 0 {
			marker = "*" // Current window
		} else if i == 1 {
			marker = ">" // Toggle target (previous window)
		}

		appID := window.AppID
		if appID == "" {
			appID = window.Class
		}
		if appID == "" {
			appID = "<unknown>"
		}

		fmt.Printf("%s [%d] %s - %s\n", marker, window.ID, window.Name, appID)
	}

	fmt.Println("\nLegend: * = current window, > = toggle target")
}

var stackClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the window stack",
	Run: runStackClear,
}

func runStackClear(cmd *cobra.Command, args []string) {
	log := logger.GetDefaultLogger()
	log.LogDebug("Stack clear requested")

	// Connect to daemon
	client := ipc.NewDaemonClient()
	socketPath := getSocketPath()

	if err := client.Connect(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure the daemon is running: sway-compat daemon\n")
		os.Exit(1)
	}
	defer client.Close()

	// Send clear request
	if err := client.Clear(); err != nil {
		log.LogError("Failed to clear stack", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.LogInfo("Stack cleared successfully")
	fmt.Println("Window stack cleared successfully")
}

func init() {
	rootCmd.AddCommand(stackCmd)
	stackCmd.AddCommand(stackToggleCmd)
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackClearCmd)
}
