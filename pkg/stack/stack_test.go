package stack

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cristianoliveira/sway-compat/internal/logger"
	"github.com/cristianoliveira/sway-compat/pkg/ipc"
)

func init() {
	// Initialize empty logger for tests
	logger.SetDefaultLogger(&logger.EmptyLogger{})
}

// MockIPCClient is a mock implementation of ipc.Manager
type MockIPCClient struct {
	windows      map[int64]*ipc.WindowInfo
	focusedID    int64
	focusError   error
	getTreeError error
	mu           sync.RWMutex
}

func NewMockIPCClient() *MockIPCClient {
	return &MockIPCClient{
		windows: make(map[int64]*ipc.WindowInfo),
	}
}

func (m *MockIPCClient) AddWindow(w ipc.WindowInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.windows[w.ID] = &w
}

func (m *MockIPCClient) RemoveWindow(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.windows, id)
}

func (m *MockIPCClient) SetFocused(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.focusedID = id
}

func (m *MockIPCClient) Connect() error {
	return nil
}

func (m *MockIPCClient) GetTree() (*ipc.Tree, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getTreeError != nil {
		return nil, m.getTreeError
	}

	// Build a simple tree with all windows as children
	tree := &ipc.Tree{
		ID:    1,
		Name:  "root",
		Type:  "root",
		Nodes: make([]*ipc.Tree, 0, len(m.windows)),
	}

	for _, w := range m.windows {
		tree.Nodes = append(tree.Nodes, &ipc.Tree{
			ID:       w.ID,
			Name:     w.Name,
			AppID:    w.AppID,
			Class:    w.Class,
			Instance: w.Instance,
			Type:     "con",
			Focused:  w.Focused,
		})
	}

	return tree, nil
}

func (m *MockIPCClient) GetFocusedWindow() (*ipc.WindowInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.focusedID == 0 {
		return nil, fmt.Errorf("no focused window")
	}

	w, exists := m.windows[m.focusedID]
	if !exists {
		return nil, fmt.Errorf("focused window not found")
	}

	return w, nil
}

func (m *MockIPCClient) GetWindowInfo(id int64) (*ipc.WindowInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w, exists := m.windows[id]
	if !exists {
		return nil, fmt.Errorf("window %d not found", id)
	}

	return w, nil
}

func (m *MockIPCClient) FocusWindow(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.focusError != nil {
		return m.focusError
	}

	if _, exists := m.windows[id]; !exists {
		return fmt.Errorf("window %d does not exist", id)
	}

	m.focusedID = id
	return nil
}

func (m *MockIPCClient) RunCommand(cmd string) error {
	return nil
}

func (m *MockIPCClient) Subscribe(events []string) (chan ipc.Event, error) {
	return make(chan ipc.Event), nil
}

func (m *MockIPCClient) Close() error {
	return nil
}

// MockStorage is a mock implementation of storage.StackStorage
type MockStorage struct {
	stack      []int64
	pushError  error
	getError   error
	clearError error
	mu         sync.RWMutex
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		stack: make([]int64, 0),
	}
}

func (m *MockStorage) PushStack(windowID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pushError != nil {
		return m.pushError
	}

	// Remove duplicates
	newStack := make([]int64, 0)
	for _, id := range m.stack {
		if id != windowID {
			newStack = append(newStack, id)
		}
	}

	// Prepend new ID
	m.stack = append([]int64{windowID}, newStack...)

	return nil
}

func (m *MockStorage) GetStack() ([]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	result := make([]int64, len(m.stack))
	copy(result, m.stack)
	return result, nil
}

func (m *MockStorage) ClearStack() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clearError != nil {
		return m.clearError
	}

	m.stack = make([]int64, 0)
	return nil
}

// Generic Storage interface methods (not used in stack manager)
func (m *MockStorage) Open(path string) error                        { return nil }
func (m *MockStorage) OpenReadOnly(path string) error                { return nil }
func (m *MockStorage) Close() error                                  { return nil }
func (m *MockStorage) Get(bucket, key string) ([]byte, error)        { return nil, nil }
func (m *MockStorage) Put(bucket, key string, value []byte) error    { return nil }
func (m *MockStorage) Delete(bucket, key string) error               { return nil }
func (m *MockStorage) List(bucket string) (map[string][]byte, error) { return nil, nil }
func (m *MockStorage) CreateBucket(bucket string) error              { return nil }

// Tests

func TestStackManager_Push_Single(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	window := ipc.WindowInfo{
		ID:    100,
		Name:  "Test Window",
		AppID: "test-app",
	}

	manager.Push(window)

	// Verify in-memory stack
	if manager.Size() != 1 {
		t.Errorf("Expected stack size 1, got %d", manager.Size())
	}

	peek, ok := manager.Peek()
	if !ok {
		t.Fatal("Expected to peek window")
	}

	if peek.ID != 100 {
		t.Errorf("Expected window ID 100, got %d", peek.ID)
	}

	// Verify storage was called
	storageStack, _ := storage.GetStack()
	if len(storageStack) != 1 || storageStack[0] != 100 {
		t.Error("Storage not updated correctly")
	}
}

func TestStackManager_Push_Multiple(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	windows := []ipc.WindowInfo{
		{ID: 100, Name: "Window 1", AppID: "app1"},
		{ID: 200, Name: "Window 2", AppID: "app2"},
		{ID: 300, Name: "Window 3", AppID: "app3"},
	}

	for _, w := range windows {
		manager.Push(w)
	}

	if manager.Size() != 3 {
		t.Errorf("Expected stack size 3, got %d", manager.Size())
	}

	// Most recent should be at front
	peek, _ := manager.Peek()
	if peek.ID != 300 {
		t.Errorf("Expected most recent window ID 300, got %d", peek.ID)
	}

	// Check order
	list := manager.List()
	expectedOrder := []int64{300, 200, 100}
	for i, expected := range expectedOrder {
		if list[i].ID != expected {
			t.Errorf("Expected ID %d at index %d, got %d", expected, i, list[i].ID)
		}
	}
}

func TestStackManager_Push_Deduplication(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Window 3", AppID: "app3"}

	manager.Push(w1)
	manager.Push(w2)
	manager.Push(w3)

	// Push w2 again (should move to front)
	manager.Push(w2)

	// Should still have 3 items
	if manager.Size() != 3 {
		t.Errorf("Expected stack size 3 after deduplication, got %d", manager.Size())
	}

	// w2 should be at front
	peek, _ := manager.Peek()
	if peek.ID != 200 {
		t.Errorf("Expected window ID 200 at front, got %d", peek.ID)
	}

	// Check order: 200, 300, 100
	list := manager.List()
	expectedOrder := []int64{200, 300, 100}
	for i, expected := range expectedOrder {
		if list[i].ID != expected {
			t.Errorf("Expected ID %d at index %d, got %d", expected, i, list[i].ID)
		}
	}
}

func TestStackManager_Push_MaxSize(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 5} // Small max size

	manager := NewStackManager(ipcClient, storage, config)

	// Push 10 windows
	for i := 1; i <= 10; i++ {
		manager.Push(ipc.WindowInfo{
			ID:    int64(i * 100),
			Name:  fmt.Sprintf("Window %d", i),
			AppID: fmt.Sprintf("app%d", i),
		})
	}

	// Should only have 5 items
	if manager.Size() != 5 {
		t.Errorf("Expected stack size 5 (max), got %d", manager.Size())
	}

	// Most recent 5 should be: 1000, 900, 800, 700, 600
	list := manager.List()
	expectedOrder := []int64{1000, 900, 800, 700, 600}
	for i, expected := range expectedOrder {
		if list[i].ID != expected {
			t.Errorf("Expected ID %d at index %d, got %d", expected, i, list[i].ID)
		}
	}
}

func TestStackManager_Push_ExcludeApps(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{
		StackSize:   20,
		ExcludeApps: []string{"excluded-app"},
	}

	manager := NewStackManager(ipcClient, storage, config)

	w1 := ipc.WindowInfo{ID: 100, Name: "Normal", AppID: "normal-app"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Excluded", AppID: "excluded-app"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Normal 2", AppID: "normal-app-2"}

	manager.Push(w1)
	manager.Push(w2) // Should be excluded
	manager.Push(w3)

	// Should only have 2 items (w2 excluded)
	if manager.Size() != 2 {
		t.Errorf("Expected stack size 2 (excluded 1), got %d", manager.Size())
	}

	// Verify excluded window not in stack
	list := manager.List()
	for _, w := range list {
		if w.ID == 200 {
			t.Error("Excluded window should not be in stack")
		}
	}
}

func TestStackManager_Push_IncludeOnly(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{
		StackSize:   20,
		IncludeOnly: []string{"allowed-app", "another-allowed"},
	}

	manager := NewStackManager(ipcClient, storage, config)

	w1 := ipc.WindowInfo{ID: 100, Name: "Allowed", AppID: "allowed-app"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Not Allowed", AppID: "other-app"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Also Allowed", AppID: "another-allowed"}

	manager.Push(w1)
	manager.Push(w2) // Should be excluded (not in IncludeOnly)
	manager.Push(w3)

	// Should only have 2 items
	if manager.Size() != 2 {
		t.Errorf("Expected stack size 2, got %d", manager.Size())
	}

	// Verify only allowed apps in stack
	list := manager.List()
	expectedIDs := []int64{300, 100}
	for i, expected := range expectedIDs {
		if list[i].ID != expected {
			t.Errorf("Expected ID %d at index %d, got %d", expected, i, list[i].ID)
		}
	}
}

func TestStackManager_Toggle_Success(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	// Add windows to IPC client
	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1", Focused: true}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2", Focused: false}
	w3 := ipc.WindowInfo{ID: 300, Name: "Window 3", AppID: "app3", Focused: false}

	ipcClient.AddWindow(w1)
	ipcClient.AddWindow(w2)
	ipcClient.AddWindow(w3)

	manager := NewStackManager(ipcClient, storage, config)

	// Build stack: 100, 200, 300
	manager.Push(w3)
	manager.Push(w2)
	manager.Push(w1)

	// Toggle should switch to w2 (index 1)
	toggled, ok := manager.Toggle()
	if !ok {
		t.Fatal("Toggle failed")
	}

	if toggled.ID != 200 {
		t.Errorf("Expected toggled window ID 200, got %d", toggled.ID)
	}

	// Verify IPC client focused the window
	if ipcClient.focusedID != 200 {
		t.Errorf("Expected IPC to focus window 200, got %d", ipcClient.focusedID)
	}
}

func TestStackManager_Toggle_EmptyStack(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	// Toggle on empty stack should fail
	_, ok := manager.Toggle()
	if ok {
		t.Error("Toggle should fail on empty stack")
	}
}

func TestStackManager_Toggle_SingleWindow(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	ipcClient.AddWindow(w1)

	manager := NewStackManager(ipcClient, storage, config)
	manager.Push(w1)

	// Toggle with single window should fail
	_, ok := manager.Toggle()
	if ok {
		t.Error("Toggle should fail with single window")
	}
}

func TestStackManager_Toggle_InvalidWindow(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Window 3", AppID: "app3"}

	ipcClient.AddWindow(w1)
	ipcClient.AddWindow(w2)
	ipcClient.AddWindow(w3)

	manager := NewStackManager(ipcClient, storage, config)

	// Stack order after pushes: [300, 200, 100]
	manager.Push(w1)
	manager.Push(w2)
	manager.Push(w3)

	// Remove w2 from IPC (simulate window closed)
	ipcClient.RemoveWindow(200)

	// Toggle should skip invalid w2 at index 1, remove it from stack
	// New stack becomes: [300, 100]
	// Then focus window at new index 1, which is 100
	toggled, ok := manager.Toggle()
	if !ok {
		t.Fatal("Toggle should succeed by skipping invalid window")
	}

	if toggled.ID != 100 {
		t.Errorf("Expected toggled to window 100 (skipped invalid 200), got %d", toggled.ID)
	}
}

func TestStackManager_PeekPrevious(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}

	manager.Push(w1)
	manager.Push(w2)

	// PeekPrevious should return w1
	prev, ok := manager.PeekPrevious()
	if !ok {
		t.Fatal("PeekPrevious failed")
	}

	if prev.ID != 100 {
		t.Errorf("Expected previous window ID 100, got %d", prev.ID)
	}
}

func TestStackManager_Pop(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}

	manager.Push(w1)
	manager.Push(w2)

	// Pop should remove and return w2
	popped, ok := manager.Pop()
	if !ok {
		t.Fatal("Pop failed")
	}

	if popped.ID != 200 {
		t.Errorf("Expected popped window ID 200, got %d", popped.ID)
	}

	// Size should be 1
	if manager.Size() != 1 {
		t.Errorf("Expected size 1 after pop, got %d", manager.Size())
	}

	// w1 should be at front now
	peek, _ := manager.Peek()
	if peek.ID != 100 {
		t.Errorf("Expected window ID 100 at front, got %d", peek.ID)
	}
}

func TestStackManager_Clear(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	manager := NewStackManager(ipcClient, storage, config)

	// Add some windows
	for i := 1; i <= 5; i++ {
		manager.Push(ipc.WindowInfo{
			ID:    int64(i * 100),
			Name:  fmt.Sprintf("Window %d", i),
			AppID: fmt.Sprintf("app%d", i),
		})
	}

	// Clear
	manager.Clear()

	// Size should be 0
	if manager.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", manager.Size())
	}

	// Storage should also be cleared
	storageStack, _ := storage.GetStack()
	if len(storageStack) != 0 {
		t.Error("Storage not cleared")
	}
}

func TestStackManager_Save_Load(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	// Add windows to IPC client
	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Window 3", AppID: "app3"}

	ipcClient.AddWindow(w1)
	ipcClient.AddWindow(w2)
	ipcClient.AddWindow(w3)

	manager := NewStackManager(ipcClient, storage, config)

	// Build stack
	manager.Push(w1)
	manager.Push(w2)
	manager.Push(w3)

	// Save
	err := manager.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create new manager and load
	manager2 := NewStackManager(ipcClient, storage, config)
	err = manager2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded stack matches
	if manager2.Size() != manager.Size() {
		t.Errorf("Expected size %d after load, got %d", manager.Size(), manager2.Size())
	}

	list1 := manager.List()
	list2 := manager2.List()

	for i := range list1 {
		if list1[i].ID != list2[i].ID {
			t.Errorf("Window mismatch at index %d: expected %d, got %d", i, list1[i].ID, list2[i].ID)
		}
	}
}

func TestStackManager_Load_SkipsClosedWindows(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 20}

	// Add windows to IPC client
	w1 := ipc.WindowInfo{ID: 100, Name: "Window 1", AppID: "app1"}
	w2 := ipc.WindowInfo{ID: 200, Name: "Window 2", AppID: "app2"}
	w3 := ipc.WindowInfo{ID: 300, Name: "Window 3", AppID: "app3"}

	ipcClient.AddWindow(w1)
	ipcClient.AddWindow(w2)
	ipcClient.AddWindow(w3)

	// Manually add IDs to storage
	storage.PushStack(300)
	storage.PushStack(200)
	storage.PushStack(100)

	// Remove w2 from IPC (simulate closed window)
	ipcClient.RemoveWindow(200)

	// Load should skip w2
	manager := NewStackManager(ipcClient, storage, config)
	err := manager.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should only have w1 and w3
	if manager.Size() != 2 {
		t.Errorf("Expected size 2 (skipped closed window), got %d", manager.Size())
	}

	list := manager.List()
	expectedIDs := []int64{100, 300}
	for i, expected := range expectedIDs {
		if list[i].ID != expected {
			t.Errorf("Expected ID %d at index %d, got %d", expected, i, list[i].ID)
		}
	}
}

func TestStackManager_ThreadSafety(t *testing.T) {
	ipcClient := NewMockIPCClient()
	storage := NewMockStorage()
	config := Config{StackSize: 100}

	manager := NewStackManager(ipcClient, storage, config)

	// Concurrent pushes
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			manager.Push(ipc.WindowInfo{
				ID:    int64(id),
				Name:  fmt.Sprintf("Window %d", id),
				AppID: fmt.Sprintf("app%d", id),
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.List()
			manager.Size()
			manager.Peek()
		}()
	}

	wg.Wait()

	// Should have all windows (or up to max size)
	size := manager.Size()
	if size == 0 {
		t.Error("Expected non-zero size after concurrent operations")
	}
}
