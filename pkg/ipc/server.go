package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/cristianoliveira/sway-compat/internal/logger"
)

// StackManager interface defines methods needed by the IPC server
type StackManager interface {
	Toggle() (WindowInfo, bool)
	List() []WindowInfo
	Clear()
	Push(window WindowInfo)
}

// Server handles IPC requests from commands
type Server struct {
	socketPath string
	listener   net.Listener
	manager    StackManager
	log        logger.Logger
	mu         sync.Mutex
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

// NewServer creates a new IPC server
func NewServer(manager StackManager) *Server {
	return &Server{
		manager:  manager,
		log:      logger.GetDefaultLogger(),
		shutdown: make(chan struct{}),
	}
}

// Start starts the IPC server and listens for connections
func (s *Server) Start(socketPath string) error {
	s.socketPath = socketPath

	// Remove existing socket file if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket listener: %w", err)
	}

	s.listener = listener
	s.log.LogInfo("IPC server started", "socket", socketPath)

	// Accept connections in a loop
	for {
		select {
		case <-s.shutdown:
			s.log.LogInfo("IPC server shutting down")
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-s.shutdown:
					return nil
				default:
					s.log.LogError("Failed to accept connection", "error", err)
					continue
				}
			}

			// Handle connection in goroutine
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handleConnection(conn)
			}()
		}
	}
}

// Stop stops the server and cleans up
func (s *Server) Stop() error {
	s.log.LogInfo("Stopping IPC server")

	// Signal shutdown
	close(s.shutdown)

	// Close listener to unblock Accept()
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to finish
	s.wg.Wait()

	// Remove socket file
	if s.socketPath != "" {
		os.RemoveAll(s.socketPath)
	}

	s.log.LogInfo("IPC server stopped")
	return nil
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.log.LogDebug("New IPC connection")

	// Read request
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		s.log.LogError("Failed to read request", "error", err)
		s.sendError(conn, fmt.Sprintf("failed to read request: %v", err))
		return
	}

	// Parse request
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		s.log.LogError("Failed to unmarshal request", "error", err)
		s.sendError(conn, fmt.Sprintf("invalid request format: %v", err))
		return
	}

	s.log.LogDebug("Received IPC request", "type", req.Type)

	// Handle request
	resp := s.handleRequest(req)

	// Send response
	if err := s.sendResponse(conn, resp); err != nil {
		s.log.LogError("Failed to send response", "error", err)
	}
}

// handleRequest processes a request and returns a response
func (s *Server) handleRequest(req Request) Response {
	switch req.Type {
	case RequestTypeToggle:
		return s.handleToggle()

	case RequestTypeList:
		return s.handleList()

	case RequestTypeClear:
		return s.handleClear()

	case RequestTypePush:
		return s.handlePush(req)

	default:
		return Response{
			Success: false,
			Error:   fmt.Sprintf("unknown request type: %s", req.Type),
		}
	}
}

// handleToggle handles TOGGLE requests
func (s *Server) handleToggle() Response {
	window, ok := s.manager.Toggle()
	if !ok {
		return Response{
			Success: false,
			Error:   "not enough windows in stack to toggle (need at least 2)",
		}
	}

	return Response{
		Success: true,
		Window:  &window,
	}
}

// handleList handles LIST requests
func (s *Server) handleList() Response {
	windows := s.manager.List()

	return Response{
		Success: true,
		Windows: windows,
	}
}

// handleClear handles CLEAR requests
func (s *Server) handleClear() Response {
	s.manager.Clear()

	return Response{
		Success: true,
	}
}

// handlePush handles PUSH requests (for testing/manual control)
func (s *Server) handlePush(req Request) Response {
	if req.WindowID == nil {
		return Response{
			Success: false,
			Error:   "window_id is required for PUSH request",
		}
	}

	// Create minimal WindowInfo for push
	window := WindowInfo{
		ID: *req.WindowID,
	}

	s.manager.Push(window)

	return Response{
		Success: true,
	}
}

// sendResponse sends a response to the client
func (s *Server) sendResponse(conn net.Conn, resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Append newline for line-delimited JSON
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// sendError sends an error response to the client
func (s *Server) sendError(conn net.Conn, errMsg string) {
	resp := Response{
		Success: false,
		Error:   errMsg,
	}
	s.sendResponse(conn, resp)
}
