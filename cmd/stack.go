package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/cristianoliveira/sway-compat/pkg/stack"
	"github.com/cristianoliveira/sway-compat/pkg/storage"
	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage window focus stack",
	Long: `Window stack management for Alt+Tab style switching.
Maintains a history of focused windows and allows toggling between them.`,
}

var (
	daemonDBPath     string
	daemonStackSize  int
	daemonExcludeApps []string
	daemonIncludeOnly []string
)

var stackDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run stack tracking daemon in background",
	Long: `Start the stack tracking daemon that listens to Sway IPC events
and maintains the window focus history.`,
	Run: runStackDaemon,
}

func runStackDaemon(cmd *cobra.Command, args []string) {
	log := logger.GetDefaultLogger()
	log.LogInfo("Starting stack tracking daemon")

	// Expand home directory in DB path
	dbPath := daemonDBPath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(home, ".local", "state", "sway-compat-stack.json")
	}

	// Initialize IPC client
	client, err := ipc.NewClient()
	if err != nil {
		log.LogError("Failed to connect to Sway IPC", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize storage - will be opened/closed on each operation
	storage := storage.NewFileStorage()

	// Initialize stack manager
	config := stack.Config{
		StackSize:   daemonStackSize,
		DBPath:      dbPath,
		ExcludeApps: daemonExcludeApps,
		IncludeOnly: daemonIncludeOnly,
	}
	manager := stack.NewStackManager(client, storage, config)

	// Set storage path
	if err := storage.Open(dbPath); err != nil {
		log.LogError("Failed to set storage path", "error", err)
	}

	// Load existing stack from storage
	if err := manager.Load(); err != nil {
		log.LogError("Failed to load stack from storage", "error", err)
		// Continue anyway - not fatal
	}

	// Subscribe to window events
	eventChan, err := client.Subscribe([]string{"window"})
	if err != nil {
		log.LogError("Failed to subscribe to events", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to subscribe to window events: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.LogInfo("Stack daemon started successfully",
		"db_path", dbPath,
		"stack_size", daemonStackSize,
		"exclude_apps", daemonExcludeApps,
		"include_only", daemonIncludeOnly)

	fmt.Println("Stack daemon started. Press Ctrl+C to stop.")

	// Event loop
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				log.LogInfo("Event channel closed, shutting down")
				fmt.Println("\nEvent channel closed")
				return
			}

			// Only track "focus" change events
			if event.Change == "focus" && event.Container != nil {
				log.LogDebug("Focus change detected",
					"window_id", event.Container.ID,
					"window_name", event.Container.Name,
					"app_id", event.Container.AppID)

				// Push to stack
				manager.Push(*event.Container)
			}

		case sig := <-sigChan:
			log.LogInfo("Received signal, shutting down", "signal", sig)
			fmt.Printf("\nReceived %v, shutting down gracefully...\n", sig)

			// Save stack before exiting
			if err := manager.Save(); err != nil {
				log.LogError("Failed to save stack on shutdown", "error", err)
			}

			return
		}
	}
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

	// Get DB path
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(home, ".local", "state", "sway-compat-stack.json")

	// Initialize IPC client
	client, err := ipc.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize storage and stack manager
	storage := storage.NewFileStorage()
	config := stack.Config{StackSize: 20, DBPath: dbPath}
	manager := stack.NewStackManager(client, storage, config)

	// Set storage path
	if err := storage.Open(dbPath); err != nil {
		log.LogError("Failed to set storage path", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to set storage path: %v\n", err)
		os.Exit(1)
	}

	// Load stack from storage (FileStorage handles locking)
	if err := manager.Load(); err != nil {
		log.LogError("Failed to load stack", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to load stack: %v\n", err)
		os.Exit(1)
	}

	// Toggle to previous window
	window, ok := manager.Toggle()
	if !ok {
		log.LogDebug("Toggle failed - not enough windows in stack")
		fmt.Fprintf(os.Stderr, "Error: Not enough windows in stack to toggle (need at least 2)\n")
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

	// Get DB path
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(home, ".local", "state", "sway-compat-stack.json")

	// Initialize IPC client
	client, err := ipc.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize storage and stack manager
	storage := storage.NewFileStorage()
	config := stack.Config{StackSize: 20, DBPath: dbPath}
	manager := stack.NewStackManager(client, storage, config)

	// Set storage path
	if err := storage.Open(dbPath); err != nil {
		log.LogError("Failed to set storage path", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to set storage path: %v\n", err)
		os.Exit(1)
	}

	// Load stack from storage (FileStorage handles locking)
	if err := manager.Load(); err != nil {
		log.LogError("Failed to load stack", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to load stack: %v\n", err)
		os.Exit(1)
	}

	// Get stack
	windows := manager.List()

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

	// Get DB path
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(home, ".local", "state", "sway-compat-stack.json")

	// Initialize storage (no need for IPC client for clear)
	storage := storage.NewFileStorage()
	if err := storage.Open(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open storage: %v\n", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Clear the stack
	if err := storage.ClearStack(); err != nil {
		log.LogError("Failed to clear stack", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to clear stack: %v\n", err)
		os.Exit(1)
	}

	log.LogInfo("Stack cleared successfully")
	fmt.Println("Window stack cleared successfully")
}

func init() {
	rootCmd.AddCommand(stackCmd)
	stackCmd.AddCommand(stackDaemonCmd)
	stackCmd.AddCommand(stackToggleCmd)
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackClearCmd)

	// Daemon flags
	stackDaemonCmd.Flags().StringVar(&daemonDBPath, "db-path", "",
		"Path to stack file (default: ~/.local/state/sway-compat-stack.json)")
	stackDaemonCmd.Flags().IntVar(&daemonStackSize, "stack-size", 20,
		"Maximum number of windows to track in stack")
	stackDaemonCmd.Flags().StringSliceVar(&daemonExcludeApps, "exclude", []string{},
		"App IDs to exclude from stack (comma-separated)")
	stackDaemonCmd.Flags().StringSliceVar(&daemonIncludeOnly, "include-only", []string{},
		"Only track these app IDs (comma-separated, empty means track all)")
}
