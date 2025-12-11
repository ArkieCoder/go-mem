package scoring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONFileStorage_SaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "go-mem-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Define a custom path for the storage
	testPath := filepath.Join(tmpDir, "scores.json")

	// Initialize storage with the custom path manually
	// (Since NewJSONFileStorage uses user home dir, we bypass it or assume we can struct init)
	// struct is exported? No, JSONFileStorage is exported. Fields? path is unexported.
	// Check storage.go: type JSONFileStorage struct { path string }
	// We need a way to create it with a custom path or use NewJSONFileStorage and mock UserHomeDir?
	// mocking os.UserHomeDir is hard in Go without a wrapper.

	// Let's modify storage.go to allow injecting the path or just create a new constructor for testing?
	// Or we can rely on `NewJSONFileStorage` but that writes to real disk. Not good.
	// Best approach: Modifying storage.go to verify we can set the path, OR
	// Just duplicate the struct initialization here since it's in the same package (internal/scoring).

	storage := &JSONFileStorage{path: testPath}

	// 1. Test Load on non-existent file (should return empty)
	entries, err := storage.LoadAll()
	if err != nil {
		t.Errorf("LoadAll on non-existent file returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}

	// 2. Test Save
	testEntries := []ScoreHistoryEntry{
		{Hash: "abc", Score: 100, Title: "Test1", Timestamp: "2023-01-01"},
		{Hash: "def", Score: 200, Title: "Test2", Timestamp: "2023-01-02"},
	}

	err = storage.SaveAll(testEntries)
	if err != nil {
		t.Fatalf("SaveAll returned error: %v", err)
	}

	// Verify file existence
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("File was not created at %s", testPath)
	}

	// 3. Test Load again (should return saved entries)
	loadedEntries, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}

	if len(loadedEntries) != len(testEntries) {
		t.Errorf("Expected %d entries, got %d", len(testEntries), len(loadedEntries))
	}

	// Check content
	if loadedEntries[0].Hash != "abc" || loadedEntries[1].Score != 200 {
		t.Errorf("Loaded content mismatch. Got: %+v", loadedEntries)
	}
}

func TestJSONFileStorage_CorruptFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-mem-test-corrupt")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPath := filepath.Join(tmpDir, "corrupt.json")

	// Write garbage to file
	err = os.WriteFile(testPath, []byte("{ not valid json }"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	storage := &JSONFileStorage{path: testPath}

	_, err = storage.LoadAll()
	if err == nil {
		t.Error("Expected error when loading corrupt file, got nil")
	}
}

func TestJSONFileStorage_EmptyFile(t *testing.T) {
	// Empty file is valid (EOF handled)
	tmpDir, err := os.MkdirTemp("", "go-mem-test-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPath := filepath.Join(tmpDir, "empty.json")

	// Write empty file
	err = os.WriteFile(testPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	storage := &JSONFileStorage{path: testPath}

	entries, err := storage.LoadAll()
	if err != nil {
		t.Errorf("LoadAll on empty file returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries from empty file, got %d", len(entries))
	}
}
