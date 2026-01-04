package ipc

import (
	"encoding/json"
	"fmt"

	"codeberg.org/scip/swayipc/v2"
	"github.com/cristianoliveira/sway-compat/internal/logger"
)

// Client implements the Manager interface using swayipc
type Client struct {
	commandClient *swayipc.SwayIPC
	eventClient   *swayipc.SwayIPC
	log           logger.Logger
}

// NewClient creates a new IPC client
func NewClient() (*Client, error) {
	log := logger.GetDefaultLogger()
	log.LogDebug("Connecting to Sway IPC")

	commandClient := swayipc.NewSwayIPC()
	if err := commandClient.Connect(); err != nil {
		log.LogError("Failed to connect to Sway IPC", "error", err)
		return nil, fmt.Errorf("failed to connect to sway: %w", err)
	}

	log.LogInfo("Successfully connected to Sway IPC")
	return &Client{commandClient: commandClient, log: log}, nil
}

// Connect establishes connection to Sway IPC socket
func (c *Client) Connect() error {
	if c.commandClient != nil {
		return nil
	}

	c.commandClient = swayipc.NewSwayIPC()
	if err := c.commandClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to sway: %w", err)
	}

	return nil
}

// GetTree retrieves the window tree from Sway
func (c *Client) GetTree() (*Tree, error) {
	tree, err := c.commandClient.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}
	return convertNode(tree), nil
}

// GetFocusedWindow returns the currently focused window
func (c *Client) GetFocusedWindow() (*WindowInfo, error) {
	c.log.LogDebug("Getting focused window")

	tree, err := c.commandClient.GetTree()
	if err != nil {
		c.log.LogError("Failed to get tree", "error", err)
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	focusedNode := tree.FindFocused()
	if focusedNode == nil {
		c.log.LogError("No focused window found")
		return nil, fmt.Errorf("no focused window found")
	}

	// Convert to WindowInfo
	info := &WindowInfo{
		ID:      int64(focusedNode.Id),
		Name:    focusedNode.Name,
		AppID:   focusedNode.X11Window,
		Type:    focusedNode.Type,
		Focused: focusedNode.Focused,
		Visible: focusedNode.Visible,
	}

	c.log.LogDebug("Found focused window",
		"id", info.ID,
		"name", info.Name,
		"app_id", info.AppID,
		"type", info.Type)

	return info, nil
}

// FocusWindow focuses the window with the given ID
func (c *Client) FocusWindow(id int64) error {
	cmd := fmt.Sprintf("[con_id=%d] focus", id)
	_, err := c.commandClient.RunGlobalCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to focus window %d: %w", id, err)
	}
	return nil
}

// Subscribe subscribes to Sway events
func (c *Client) Subscribe(events []string) (chan Event, error) {
	c.log.LogDebug("Subscribing to Sway events", "events", events)

	// Event subscriptions need their own connection. Once a connection is
	// subscribed it can no longer be used for other IPC commands, so we keep a
	// dedicated client just for events.
	if c.eventClient == nil {
		c.eventClient = swayipc.NewSwayIPC()
		if err := c.eventClient.Connect(); err != nil {
			c.log.LogError("Failed to connect Sway IPC for events", "error", err)
			return nil, fmt.Errorf("failed to connect to sway for events: %w", err)
		}
	}

	sub := &swayipc.Event{}
	for _, e := range events {
		switch e {
		case "window":
			sub.Window = true
		case "workspace":
			sub.Workspace = true
		default:
			return nil, fmt.Errorf("unsupported event type: %s", e)
		}
	}

	_, err := c.eventClient.Subscribe(sub)
	if err != nil {
		c.log.LogError("Failed to subscribe to events", "error", err)
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	eventChan := make(chan Event, 10)

	// Start goroutine to run event loop
	go func() {
		defer close(eventChan)
		c.log.LogDebug("Event loop started")

		// Run the event loop with callback
		err := c.eventClient.EventLoop(func(rawEvent *swayipc.RawResponse) error {
			// Only process window events for now
			// PayloadType 0x80000003 is WINDOW event
			// We'll unmarshal based on what was subscribed

			var windowEvent swayipc.EventWindow
			if err := json.Unmarshal(rawEvent.Payload, &windowEvent); err != nil {
				c.log.LogError("Failed to unmarshal event", "error", err)
				return nil // Don't stop event loop on unmarshal errors
			}

			// Convert to our Event type
			event := Event{
				Change: windowEvent.Change,
			}

			// Convert container if present
			if windowEvent.Container != nil {
				event.Container = &WindowInfo{
					ID:      int64(windowEvent.Container.Id),
					Name:    windowEvent.Container.Name,
					AppID:   windowEvent.Container.X11Window, // X11Window is actually app_id
					Type:    windowEvent.Container.Type,
					Focused: windowEvent.Container.Focused,
					Visible: windowEvent.Container.Visible,
				}
			}

			c.log.LogDebug("Received event",
				"change", event.Change,
				"container_id", func() int64 {
					if event.Container != nil {
						return event.Container.ID
					}
					return 0
				}())

			// Forward event to our channel (non-blocking)
			select {
			case eventChan <- event:
				// Event sent successfully
			default:
				// Channel buffer full, log warning
				c.log.LogError("Event channel full, dropping event", "change", event.Change)
			}

			return nil // Continue event loop
		})

		if err != nil {
			c.log.LogError("Event loop terminated with error", "error", err)
		} else {
			c.log.LogDebug("Event loop terminated normally")
		}
	}()

	c.log.LogInfo("Successfully subscribed to Sway events")
	return eventChan, nil
}

// Close closes the IPC connection
func (c *Client) Close() error {
	if c.commandClient != nil {
		c.commandClient.Close()
	}

	if c.eventClient != nil {
		c.eventClient.Close()
	}
	return nil
}

// convertNode converts swayipc.Node to our Tree type
func convertNode(node *swayipc.Node) *Tree {
	if node == nil {
		return nil
	}

	tree := &Tree{
		ID:      int64(node.Id),
		Name:    node.Name,
		Type:    node.Type,
		Focused: node.Focused,
		AppID:   node.X11Window, // X11Window is actually the app_id field
		Window:  int64(node.Window),
		Nodes:   make([]*Tree, len(node.Nodes)),
	}

	// TODO: swayipc doesn't expose class/instance separately
	// They would need to be extracted from other fields if available

	for i, child := range node.Nodes {
		tree.Nodes[i] = convertNode(child)
	}

	for _, floatingNode := range node.FloatingNodes {
		tree.Nodes = append(tree.Nodes, convertNode(floatingNode))
	}

	return tree
}

// convertNodeToWindowInfo converts swayipc.Node to WindowInfo
func convertNodeToWindowInfo(node *swayipc.Node) *WindowInfo {
	if node == nil {
		return nil
	}

	info := &WindowInfo{
		ID:      int64(node.Id),
		Name:    node.Name,
		AppID:   node.X11Window,
		Type:    node.Type,
		Focused: node.Focused,
		Visible: node.Visible,
	}

	return info
}

// findFocused recursively searches for the focused window in the tree
func findFocused(tree *Tree) *WindowInfo {
	if tree == nil {
		return nil
	}

	if (tree.Type == "con" || tree.Type == "floating_con") && tree.Focused && tree.Window > 0 {
		return &WindowInfo{
			ID:       tree.ID,
			Name:     tree.Name,
			AppID:    tree.AppID,
			Class:    tree.Class,
			Instance: tree.Instance,
			Type:     tree.Type,
			Focused:  tree.Focused,
		}
	}

	for _, child := range tree.Nodes {
		if found := findFocused(child); found != nil {
			return found
		}
	}

	return nil
}

// FindAllWindows returns all windows in the tree
func FindAllWindows(tree *Tree) []WindowInfo {
	log := logger.GetDefaultLogger()
	var windows []WindowInfo
	collectWindows(tree, &windows)
	log.LogDebug("Found windows in tree", "count", len(windows))
	return windows
}

// collectWindows recursively collects all windows from the tree
func collectWindows(tree *Tree, windows *[]WindowInfo) {
	if tree == nil {
		return
	}

	if tree.Type == "con" || tree.Type == "floating_con" {
		if tree.AppID != "" || tree.Class != "" || tree.Name != "" {
			*windows = append(*windows, WindowInfo{
				ID:       tree.ID,
				Name:     tree.Name,
				AppID:    tree.AppID,
				Class:    tree.Class,
				Instance: tree.Instance,
				Type:     tree.Type,
				Focused:  tree.Focused,
			})
		}
	}

	for _, child := range tree.Nodes {
		collectWindows(child, windows)
	}
}
