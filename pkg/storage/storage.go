package storage

// Storage defines the interface for persisting application state
type Storage interface {
	// Open opens the storage connection
	Open(path string) error

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
