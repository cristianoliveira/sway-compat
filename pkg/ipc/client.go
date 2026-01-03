package ipc

import (
	"fmt"

	"codeberg.org/scip/swayipc/v2"
)

// Client implements the Manager interface using swayipc
type Client struct {
	client *swayipc.SwayIPC
}

// NewClient creates a new IPC client
func NewClient() (*Client, error) {
	client := swayipc.NewSwayIPC()
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to sway: %w", err)
	}
	return &Client{client: client}, nil
}

// Connect establishes connection to Sway IPC socket
func (c *Client) Connect() error {
	return c.client.Connect()
}

// GetTree retrieves the window tree from Sway
func (c *Client) GetTree() (*Tree, error) {
	tree, err := c.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}
	return convertNode(tree), nil
}

// GetFocusedWindow returns the currently focused window
func (c *Client) GetFocusedWindow() (*WindowInfo, error) {
	tree, err := c.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// Use swayipc's built-in FindFocused method
	focusedNode := tree.FindFocused()
	if focusedNode == nil {
		return nil, fmt.Errorf("no focused window found")
	}

	// Convert to WindowInfo
	info := &WindowInfo{
		ID:       int64(focusedNode.Id),
		Name:     focusedNode.Name,
		AppID:    focusedNode.X11Window,
		Type:     focusedNode.Type,
		Focused:  focusedNode.Focused,
		Visible:  focusedNode.Visible,
	}

	return info, nil
}

// FocusWindow focuses the window with the given ID
func (c *Client) FocusWindow(id int64) error {
	cmd := fmt.Sprintf("[con_id=%d] focus", id)
	_, err := c.client.RunGlobalCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to focus window %d: %w", id, err)
	}
	return nil
}

// Subscribe subscribes to Sway events
func (c *Client) Subscribe(events []string) (chan Event, error) {
	// Create event subscription struct
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

	// Subscribe to events
	_, err := c.client.Subscribe(sub)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	eventChan := make(chan Event, 10)

	// Note: Full event loop implementation would go here
	// For now, return the channel (will be implemented when needed for daemon mode)

	return eventChan, nil
}

// Close closes the IPC connection
func (c *Client) Close() error {
	c.client.Close()
	return nil
}

// convertNode converts swayipc.Node to our Tree type
func convertNode(node *swayipc.Node) *Tree {
	if node == nil {
		return nil
	}

	tree := &Tree{
		ID:       int64(node.Id),
		Name:     node.Name,
		Type:     node.Type,
		Focused:  node.Focused,
		AppID:    node.X11Window, // X11Window is actually the app_id field
		Window:   int64(node.Window),
		Nodes:    make([]*Tree, len(node.Nodes)),
	}

	// Note: swayipc doesn't expose class/instance separately
	// They would need to be extracted from other fields if available

	// Recursively convert child nodes
	for i, child := range node.Nodes {
		tree.Nodes[i] = convertNode(child)
	}

	// Also include floating nodes
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

	// If this is a window (con or floating_con) and it's focused, return it
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

	// Recursively search children
	for _, child := range tree.Nodes {
		if found := findFocused(child); found != nil {
			return found
		}
	}

	return nil
}

// FindAllWindows returns all windows in the tree
func FindAllWindows(tree *Tree) []WindowInfo {
	var windows []WindowInfo
	collectWindows(tree, &windows)
	return windows
}

// collectWindows recursively collects all windows from the tree
func collectWindows(tree *Tree, windows *[]WindowInfo) {
	if tree == nil {
		return
	}

	// If this is a container with a meaningful name, add it
	// In Wayland/Sway, windows often have window=null, so we can't rely on Window ID
	// Instead, check if it's a con/floating_con with either an app_id or a name
	if (tree.Type == "con" || tree.Type == "floating_con") {
		// Real windows have at least an app_id or a name
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

	// Recursively collect from children
	for _, child := range tree.Nodes {
		collectWindows(child, windows)
	}
}
