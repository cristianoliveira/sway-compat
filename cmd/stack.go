package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage window focus stack",
	Long: `Window stack management for Alt+Tab style switching.
Maintains a history of focused windows and allows toggling between them.`,
}

var stackDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run stack tracking daemon in background",
	Long: `Start the stack tracking daemon that listens to Sway IPC events
and maintains the window focus history.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stack daemon - not yet implemented")
	},
}

var stackToggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle between current and previous window",
	Long: `Switch focus between the current window and the previously focused window.
This is typically bound to Mod+Tab for quick window switching.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stack toggle - not yet implemented")
	},
}

var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current window stack",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stack list - not yet implemented")
	},
}

var stackClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the window stack",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stack clear - not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(stackCmd)
	stackCmd.AddCommand(stackDaemonCmd)
	stackCmd.AddCommand(stackToggleCmd)
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackClearCmd)

	// Add flags for stack commands if needed
	// stackDaemonCmd.Flags().String("config", "", "Config file path")
}
