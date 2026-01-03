package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sway-compat",
	Short: "Sway window manager compatibility tools",
	Long: `A collection of utilities for Sway window manager that provide
macOS-like window cycling and stack management features.

Features:
  - Window cycling: Cycle through windows of the same application
  - Window stack: Alt+Tab style window switching with focus history`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
}
