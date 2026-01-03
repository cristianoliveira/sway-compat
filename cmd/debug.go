package cmd

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:    "debug",
	Short:  "Debug commands for troubleshooting",
	Hidden: true,
}

var debugFocusedCmd = &cobra.Command{
	Use:   "focused",
	Short: "Show information about the currently focused window",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ipc.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		focused, err := client.GetFocusedWindow()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Focused Window:")
		fmt.Printf("  ID:      %d\n", focused.ID)
		fmt.Printf("  Name:    %s\n", focused.Name)
		fmt.Printf("  App ID:  %s\n", focused.AppID)
		fmt.Printf("  Class:   %s\n", focused.Class)
		fmt.Printf("  Type:    %s\n", focused.Type)
		fmt.Printf("  Focused: %t\n", focused.Focused)
		fmt.Printf("  Visible: %t\n", focused.Visible)
	},
}

var debugTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show all windows in the tree",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ipc.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		tree, err := client.GetTree()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		windows := ipc.FindAllWindows(tree)
		fmt.Printf("Found %d windows:\n\n", len(windows))

		for i, win := range windows {
			fmt.Printf("%d. %s\n", i+1, win.Name)
			fmt.Printf("   ID:      %d\n", win.ID)
			fmt.Printf("   App ID:  %s\n", win.AppID)
			fmt.Printf("   Class:   %s\n", win.Class)
			fmt.Printf("   Type:    %s\n", win.Type)
			fmt.Printf("   Focused: %t\n", win.Focused)
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.AddCommand(debugFocusedCmd)
	debugCmd.AddCommand(debugTreeCmd)
}
