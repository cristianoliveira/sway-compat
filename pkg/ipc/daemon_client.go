package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// DaemonClient handles communication with the sway-compat daemon
type DaemonClient struct {
	conn       net.Conn
	socketPath string
	timeout    time.Duration
}

// NewDaemonClient creates a new daemon client
func NewDaemonClient() *DaemonClient {
	return &DaemonClient{
		timeout: 5 * time.Second, // Default 5-second timeout
	}
}

// Connect connects to the daemon's Unix socket
func (c *DaemonClient) Connect(socketPath string) error {
	c.socketPath = socketPath

	conn, err := net.DialTimeout("unix", socketPath, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn
	return nil
}

// Close closes the connection to the daemon
func (c *DaemonClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Toggle sends a toggle request to the daemon
func (c *DaemonClient) Toggle() (*WindowInfo, error) {
	req := Request{
		Type: RequestTypeToggle,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	if resp.Window == nil {
		return nil, fmt.Errorf("daemon returned success but no window")
	}

	return resp.Window, nil
}

// List sends a list request to the daemon
func (c *DaemonClient) List() ([]WindowInfo, error) {
	req := Request{
		Type: RequestTypeList,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.Windows, nil
}

// Clear sends a clear request to the daemon
func (c *DaemonClient) Clear() error {
	req := Request{
		Type: RequestTypeClear,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}

	return nil
}

// Push sends a push request to the daemon (for testing)
func (c *DaemonClient) Push(windowID int64) error {
	req := Request{
		Type:     RequestTypePush,
		WindowID: &windowID,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}

	return nil
}

// sendRequest sends a request to the daemon and waits for response
func (c *DaemonClient) sendRequest(req Request) (*Response, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Append newline for line-delimited JSON
	data = append(data, '\n')

	// Send request
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
