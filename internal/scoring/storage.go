package scoring

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ScoreStorage defines the interface for loading and saving score data.
// This allows for mocking the storage layer during tests.
type ScoreStorage interface {
	// LoadAll loads all score entries from the persistence layer.
	LoadAll() ([]ScoreHistoryEntry, error)
	// SaveAll saves a slice of score entries to the persistence layer, overwriting existing data.
	SaveAll(entries []ScoreHistoryEntry) error
}

// JSONFileStorage is an implementation of ScoreStorage that uses a JSON file.
type JSONFileStorage struct {
	path string
}

// NewJSONFileStorage creates a new instance of JSONFileStorage,
// automatically determining the path for the scores file.
func NewJSONFileStorage() (*JSONFileStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}
	scoreFilePath := filepath.Join(homeDir, ".config", "go-mem", "scores.json")
	return &JSONFileStorage{path: scoreFilePath}, nil
}

// LoadAll reads and decodes all score entries from the JSON file.
func (jfs *JSONFileStorage) LoadAll() ([]ScoreHistoryEntry, error) {
	file, err := os.Open(jfs.path)
	// If the file doesn't exist, it's not an error; return an empty slice.
	if os.IsNotExist(err) {
		return []ScoreHistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error opening scores file for reading: %w", err)
	}
	defer file.Close()

	entries := make([]ScoreHistoryEntry, 0)
	decoder := json.NewDecoder(file)
	// Use a loop to decode a stream of JSON objects.
	for decoder.More() {
		var entry ScoreHistoryEntry
		if err := decoder.Decode(&entry); err != nil {
			// This can happen if the file is valid but empty.
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error decoding JSON entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// SaveAll encodes and writes all score entries to the JSON file.
func (jfs *JSONFileStorage) SaveAll(entries []ScoreHistoryEntry) error {
	// Ensure the directory exists.
	dir := filepath.Dir(jfs.path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating scores directory: %w", err)
		}
	}

	file, err := os.OpenFile(jfs.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening scores file for writing: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("error encoding JSON entry: %w", err)
		}
	}

	return writer.Flush()
}
