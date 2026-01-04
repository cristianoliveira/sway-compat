package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/cristianoliveira/sway-compat/internal/logger"
)

const maxStackSize = 20

func removeID(ids []int64, targetID int64) []int64 {
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != targetID {
			result = append(result, id)
		}
	}
	return result
}


// Storage defines the interface for persisting application state
type Storage interface {
	// Open opens the storage connection
	Open(path string) error

	// OpenReadOnly opens the storage connection in read-only mode
	OpenReadOnly(path string) error

	// Close closes the storage connection
	Close() error

	// Get retrieves a value by key from the given bucket
	Get(bucket, key string) ([]byte, error)

	// Put stores a value by key in the given bucket
	Put(bucket, key string, value []byte) error

	// Delete removes a key from the given bucket
	Delete(bucket, key string) error

	// List returns all key-value pairs from the given bucket
	List(bucket string) (map[string][]byte, error)

	// CreateBucket creates a new bucket if it doesn't exist
	CreateBucket(bucket string) error
}

// StackStorage provides stack-specific storage operations
type StackStorage interface {
	// PushStack adds a window ID to the top of the stack
	PushStack(windowID int64) error

	// GetStack retrieves all window IDs in the stack
	GetStack() ([]int64, error)

	// ClearStack removes all entries from the stack
	ClearStack() error
}

// CycleStorage provides cycle-specific storage operations
type CycleStorage interface {
	// UpdateAppWindows updates the list of windows for a given app
	UpdateAppWindows(appID string, windowIDs []int64) error

	// GetAppWindows retrieves the window list for a given app
	GetAppWindows(appID string) ([]int64, error)

	// GetNextWindow calculates and returns the next window ID for cycling
	GetNextWindow(appID string, forward bool) (int64, error)

	// GetCurrentIndex returns the current index for the given app
	GetCurrentIndex(appID string) (int, error)

	// SetCurrentIndex sets the current index for the given app
	SetCurrentIndex(appID string, index int) error

	// ClearApp removes all data for the given app
	ClearApp(appID string) error
}

// FileStorage implements storage using a simple JSON file with file locking
type FileStorage struct {
	path string
	log  logger.Logger
}

// NewFileStorage creates a new file-based storage
func NewFileStorage() *FileStorage {
	return &FileStorage{
		log: logger.GetDefaultLogger(),
	}
}

// Open sets the file path (doesn't actually open the file)
func (f *FileStorage) Open(path string) error {
	// Expand home directory if present
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f.path = path
	f.log.LogInfo("File storage initialized", "path", path)
	return nil
}

// OpenReadOnly is the same as Open for file storage
func (f *FileStorage) OpenReadOnly(path string) error {
	return f.Open(path)
}

// Close is a no-op for file storage
func (f *FileStorage) Close() error {
	return nil
}

// PushStack adds a window ID to the top of the stack
func (f *FileStorage) PushStack(windowID int64) error {
	f.log.LogDebug("Pushing window to stack", "window_id", windowID)

	// Lock file for writing
	file, err := f.lockFile(true)
	if err != nil {
		return err
	}
	defer f.unlockFile(file)

	// Read current stack
	stack := f.readStackFromFile(file)

	// Remove duplicates
	stack = removeID(stack, windowID)

	// Prepend new ID
	stack = append([]int64{windowID}, stack...)

	// Trim to max size
	if len(stack) > maxStackSize {
		stack = stack[:maxStackSize]
	}

	// Write back
	return f.writeStackToFile(file, stack)
}

// GetStack retrieves all window IDs in the stack
func (f *FileStorage) GetStack() ([]int64, error) {
	f.log.LogDebug("Getting stack from file")

	// Lock file for reading
	file, err := f.lockFile(false)
	if err != nil {
		return nil, err
	}
	defer f.unlockFile(file)

	stack := f.readStackFromFile(file)
	f.log.LogDebug("Retrieved stack from file", "size", len(stack))
	return stack, nil
}

// ClearStack removes all entries from the stack
func (f *FileStorage) ClearStack() error {
	f.log.LogDebug("Clearing stack")

	// Lock file for writing
	file, err := f.lockFile(true)
	if err != nil {
		return err
	}
	defer f.unlockFile(file)

	// Write empty stack
	return f.writeStackToFile(file, []int64{})
}

// lockFile opens and locks the file (shared for reading, exclusive for writing)
func (f *FileStorage) lockFile(write bool) (*os.File, error) {
	flags := os.O_RDONLY
	if write {
		flags = os.O_RDWR | os.O_CREATE
	}

	file, err := os.OpenFile(f.path, flags, 0644)
	if err != nil {
		// If file doesn't exist for reading, that's OK (empty stack)
		if !write && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Apply file lock
	lockType := syscall.LOCK_SH // Shared lock for reading
	if write {
		lockType = syscall.LOCK_EX // Exclusive lock for writing
	}

	if err := syscall.Flock(int(file.Fd()), lockType); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to lock file: %w", err)
	}

	return file, nil
}

// unlockFile unlocks and closes the file
func (f *FileStorage) unlockFile(file *os.File) {
	if file == nil {
		return
	}
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()
}

// readStackFromFile reads the stack from a locked file
func (f *FileStorage) readStackFromFile(file *os.File) []int64 {
	if file == nil {
		return []int64{}
	}

	var stack []int64
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&stack); err != nil {
		// If file is empty or invalid, return empty stack
		return []int64{}
	}

	return stack
}

// writeStackToFile writes the stack to a locked file
func (f *FileStorage) writeStackToFile(file *os.File, stack []int64) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	// Truncate file and seek to beginning
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Write JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stack); err != nil {
		return fmt.Errorf("failed to encode stack: %w", err)
	}

	f.log.LogDebug("Stack written to file", "size", len(stack))
	return nil
}

// Generic Storage interface methods (not needed for stack)
func (f *FileStorage) Get(bucket, key string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FileStorage) Put(bucket, key string, value []byte) error {
	return fmt.Errorf("not implemented")
}

func (f *FileStorage) Delete(bucket, key string) error {
	return fmt.Errorf("not implemented")
}

func (f *FileStorage) List(bucket string) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FileStorage) CreateBucket(bucket string) error {
	return fmt.Errorf("not implemented")
}
