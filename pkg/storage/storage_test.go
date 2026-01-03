package storage

import "testing"

// Mock storage implementation for testing
type mockStorage struct {
	data map[string]map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string]map[string][]byte),
	}
}

func (m *mockStorage) Open(path string) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func (m *mockStorage) Get(bucket, key string) ([]byte, error) {
	if b, ok := m.data[bucket]; ok {
		if v, ok := b[key]; ok {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockStorage) Put(bucket, key string, value []byte) error {
	if _, ok := m.data[bucket]; !ok {
		m.data[bucket] = make(map[string][]byte)
	}
	m.data[bucket][key] = value
	return nil
}

func (m *mockStorage) Delete(bucket, key string) error {
	if b, ok := m.data[bucket]; ok {
		delete(b, key)
	}
	return nil
}

func (m *mockStorage) List(bucket string) (map[string][]byte, error) {
	if b, ok := m.data[bucket]; ok {
		return b, nil
	}
	return make(map[string][]byte), nil
}

func (m *mockStorage) CreateBucket(bucket string) error {
	if _, ok := m.data[bucket]; !ok {
		m.data[bucket] = make(map[string][]byte)
	}
	return nil
}

func TestMockStorage(t *testing.T) {
	storage := newMockStorage()

	err := storage.CreateBucket("test")
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	err = storage.Put("test", "key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	value, err := storage.Get("test", "key1")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if string(value) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(value))
	}
}
