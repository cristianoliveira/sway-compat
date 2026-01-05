package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
	"github.com/cristianoliveira/sway-compat/pkg/stack"
	"github.com/cristianoliveira/sway-compat/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	daemonDBPath      string
	daemonStackSize   int
	daemonExcludeApps []string
	daemonIncludeOnly []string

	// Scratchpad focus configuration
	daemonScratchpadFocusEnabled bool
	daemonScratchpadHideAction   string
	daemonScratchpadWorkspace    string
	daemonScratchpadDebounce     int
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run sway-compat daemon (stack tracking + IPC server)",
	Long: `Start the sway-compat daemon that:
- Listens to Sway IPC events and maintains window focus history
- Runs an IPC server for handling stack commands (toggle, list, clear)
- Persists stack state to disk periodically`,
	Run: runDaemon,
}

func runDaemon(cmd *cobra.Command, args []string) {
	log := logger.GetDefaultLogger()
	log.LogInfo("Starting sway-compat daemon")

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

	// Initialize Sway IPC client
	swayClient, err := ipc.NewClient()
	if err != nil {
		log.LogError("Failed to connect to Sway IPC", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Sway: %v\n", err)
		os.Exit(1)
	}
	defer swayClient.Close()

	// Initialize storage (for persistence)
	storage := storage.NewFileStorage()
	if err := storage.Open(dbPath); err != nil {
		log.LogError("Failed to open storage", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to open storage: %v\n", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Initialize stack manager
	config := stack.Config{
		StackSize:   daemonStackSize,
		DBPath:      dbPath,
		ExcludeApps: daemonExcludeApps,
		IncludeOnly: daemonIncludeOnly,
	}
	manager := stack.NewStackManager(swayClient, storage, config)

	// Load existing stack from storage
	if err := manager.Load(); err != nil {
		log.LogError("Failed to load stack from storage", "error", err)
		// Continue anyway - not fatal
	}

	// Create IPC server
	socketPath := getSocketPath()
	server := ipc.NewServer(manager)

	// Start IPC server in goroutine
	go func() {
		if err := server.Start(socketPath); err != nil {
			log.LogError("IPC server failed", "error", err)
		}
	}()

	// Subscribe to Sway window events
	eventChan, err := swayClient.Subscribe([]string{"window"})
	if err != nil {
		log.LogError("Failed to subscribe to events", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to subscribe to window events: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up periodic save ticker (every 30 seconds)
	saveTicker := time.NewTicker(30 * time.Second)
	defer saveTicker.Stop()

	log.LogInfo("Daemon started successfully",
		"db_path", dbPath,
		"socket_path", socketPath,
		"stack_size", daemonStackSize,
		"exclude_apps", daemonExcludeApps,
		"include_only", daemonIncludeOnly)

	fmt.Printf("Daemon started successfully.\n")
	fmt.Printf("  - IPC socket: %s\n", socketPath)
	fmt.Printf("  - Storage: %s\n", dbPath)
	fmt.Println("Press Ctrl+C to stop.")

	// Track previous window for scratchpad focus handling
	var prevWindow *ipc.WindowInfo
	var lastScratchpadAction time.Time

	// Event loop
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				log.LogInfo("Event channel closed, shutting down")
				fmt.Println("\nEvent channel closed")
				server.Stop()
				return
			}

			// Only track "focus" change events
			if event.Change == "focus" {
				if event.Container != nil {
					log.LogDebug("Focus change detected",
						"window_id", event.Container.ID,
						"window_name", event.Container.Name,
						"app_id", event.Container.AppID,
						"workspace", event.Container.Workspace,
						"scratchpad", event.Container.Scratchpad,
						"floating", event.Container.Floating)

					// Push to stack
					manager.Push(*event.Container)

					// Scratchpad focus handling
					if daemonScratchpadFocusEnabled && prevWindow != nil {
						// Debounce check
						debounceInterval := time.Duration(daemonScratchpadDebounce) * time.Millisecond
						if time.Since(lastScratchpadAction) > debounceInterval {
							handleScratchpadFocus(prevWindow, event.Container, swayClient, log)
							lastScratchpadAction = time.Now()
						}
					}

					// Update previous window
					prevWindow = event.Container
				} else {
					log.LogDebug("Focus event with no container", "change", event.Change)
				}
			}

		case <-saveTicker.C:
			// Periodic save
			log.LogDebug("Periodic save triggered")
			if err := manager.Save(); err != nil {
				log.LogError("Failed to save stack", "error", err)
			}

		case sig := <-sigChan:
			log.LogInfo("Received signal, shutting down", "signal", sig)
			fmt.Printf("\nReceived %v, shutting down gracefully...\n", sig)

			// Save stack before exiting
			if err := manager.Save(); err != nil {
				log.LogError("Failed to save stack on shutdown", "error", err)
			}

			// Stop IPC server
			server.Stop()

			return
		}
	}
}

// handleScratchpadFocus implements the scratchpad focus behavior described in docs/sway-scratchpad-focus.md
func handleScratchpadFocus(prev, curr *ipc.WindowInfo, swayClient ipc.Manager, log logger.Logger) {
	log.LogDebug("Scratchpad focus check",
		"prev_id", prev.ID,
		"prev_scratchpad", prev.Scratchpad,
		"prev_floating", prev.Floating,
		"prev_workspace", prev.Workspace,
		"curr_id", curr.ID,
		"curr_scratchpad", curr.Scratchpad,
		"curr_floating", curr.Floating,
		"curr_workspace", curr.Workspace)

	// Check if previous window is a scratchpad floating window
	prevIsScratchpad := isScratchpadWindow(prev)
	if !prevIsScratchpad {
		return
	}

	// Check if current window is non-floating (tiling) and in same workspace
	currIsFloating := curr.Floating
	if currIsFloating {
		return
	}

	// Check workspace match
	if prev.Workspace != curr.Workspace {
		return
	}

	log.LogDebug("Scratchpad focus condition met",
		"prev_id", prev.ID,
		"curr_id", curr.ID,
		"workspace", prev.Workspace)

	// First, attempt to bring current window to front by re-focusing it
	// This may not work if floating window stays on top, but we try
	if err := swayClient.FocusWindow(curr.ID); err != nil {
		log.LogError("Failed to re-focus window", "error", err, "window_id", curr.ID)
		return
	}

	// If configured to hide scratchpad, move previous window back to scratchpad
	if daemonScratchpadHideAction == "hide-scratchpad" {
		// Send move scratchpad command to Sway
		cmd := fmt.Sprintf("[con_id=%d] move scratchpad", prev.ID)
		if err := swayClient.RunCommand(cmd); err != nil {
			log.LogError("Failed to move scratchpad window", "error", err, "window_id", prev.ID, "command", cmd)
		} else {
			log.LogDebug("Moved scratchpad window back to scratchpad", "window_id", prev.ID)
		}
	}
}

// isScratchpadWindow determines if a window is a scratchpad window
func isScratchpadWindow(window *ipc.WindowInfo) bool {
	if window == nil {
		return false
	}
	// Check scratchpad_state field (requires floating)
	if window.Floating && window.Scratchpad != "" && window.Scratchpad != "none" {
		return true
	}
	// Check if floating and workspace is scratchpad workspace
	if window.Floating && window.Workspace == daemonScratchpadWorkspace {
		return true
	}
	return false
}

// getSocketPath returns the path to the Unix domain socket
func getSocketPath() string {
	// Prefer XDG_RUNTIME_DIR if available (user-specific)
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "sway-compat.sock")
	}

	// Fallback to /tmp with UID for uniqueness
	uid := os.Getuid()
	return fmt.Sprintf("/tmp/sway-compat-%d.sock", uid)
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	// Daemon flags
	daemonCmd.Flags().StringVar(&daemonDBPath, "db-path", "",
		"Path to stack file (default: ~/.local/state/sway-compat-stack.json)")
	daemonCmd.Flags().IntVar(&daemonStackSize, "stack-size", 20,
		"Maximum number of windows to track in stack")
	daemonCmd.Flags().StringSliceVar(&daemonExcludeApps, "exclude", []string{},
		"App IDs to exclude from stack (comma-separated)")
	daemonCmd.Flags().StringSliceVar(&daemonIncludeOnly, "include-only", []string{},
		"Only track these app IDs (comma-separated, empty means track all)")

	// Scratchpad focus flags
	daemonCmd.Flags().BoolVar(&daemonScratchpadFocusEnabled, "scratchpad-focus", false,
		"Enable scratchpad focus handling (auto-hide scratchpad when focusing tiling window)")
	daemonCmd.Flags().StringVar(&daemonScratchpadHideAction, "scratchpad-hide-action", "hide-scratchpad",
		"Action when scratchpad loses focus: hide-scratchpad (move to scratchpad) or noop")
	daemonCmd.Flags().StringVar(&daemonScratchpadWorkspace, "scratchpad-workspace", ".scratchpad",
		"Scratchpad workspace name (default .scratchpad)")
	daemonCmd.Flags().IntVar(&daemonScratchpadDebounce, "scratchpad-debounce", 200,
		"Debounce interval in milliseconds to avoid rapid hide/show thrash")
}
