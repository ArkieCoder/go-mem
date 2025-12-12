package scoring

import (
	"testing"
)

// MockScoreStorage is a mock implementation of the ScoreStorage interface
// that stores score entries in memory. This is used for testing.
type MockScoreStorage struct {
	Entries []ScoreHistoryEntry
	err     error // To simulate errors from the storage layer.
}

// LoadAll returns the in-memory entries or a simulated error.
func (m *MockScoreStorage) LoadAll() ([]ScoreHistoryEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.Entries, nil
}

// SaveAll replaces the in-memory entries with the provided slice or returns a simulated error.
func (m *MockScoreStorage) SaveAll(entries []ScoreHistoryEntry) error {
	if m.err != nil {
		return m.err
	}
	m.Entries = entries
	return nil
}

// TestInitScoring_NewText verifies that scoring is initialized correctly for a text
// with no prior score history.
func TestInitScoring_NewText(t *testing.T) {
	mockStorage := &MockScoreStorage{} // No history
	secret := "hello world"
	title := "Test Title"

	scoring, err := InitScoring(secret, title, mockStorage)

	if err != nil {
		t.Fatalf("InitScoring returned an unexpected error: %v", err)
	}

	if scoring.GetAttempts() != 0 {
		t.Errorf("expected 0 attempts for a new text, but got %d", scoring.GetAttempts())
	}

	if scoring.GetHighScore() != nil {
		t.Errorf("expected nil high score for a new text, but got %v", scoring.GetHighScore())
	}

	// Base score is 10 per character.
	// But CurrentScore starts at 0.
	expectedScore := 0
	if scoring.CurrentScore != expectedScore {
		t.Errorf("expected initial score of %d, but got %d", expectedScore, scoring.CurrentScore)
	}
}

// TestInitScoring_WithHistory verifies that scoring is initialized correctly for a text
// that has a previous score history.
func TestInitScoring_WithHistory(t *testing.T) {
	secret := "hello world"
	hash := calculateHash(secret)

	// Pre-populate the mock storage with a high score for our secret text
	// and another score for a different text to ensure filtering works.
	mockStorage := &MockScoreStorage{
		Entries: []ScoreHistoryEntry{
			{Hash: "some_other_hash", Score: 9999, Title: "Other"},
			{Hash: hash, Score: 500, Title: "Test Title High Score"},
			{Hash: hash, Score: 120, Title: "Test Title Low Score"},
		},
	}

	scoring, err := InitScoring(secret, "Test Title", mockStorage)

	if err != nil {
		t.Fatalf("InitScoring returned an unexpected error: %v", err)
	}

	if scoring.GetAttempts() != 2 {
		t.Errorf("expected 2 attempts, but got %d", scoring.GetAttempts())
	}

	highScore := scoring.GetHighScore()
	if highScore == nil {
		t.Fatalf("expected a high score, but got nil")
	}

	if highScore.Score != 500 {
		t.Errorf("expected high score of 500, but got %d", highScore.Score)
	}
}

// TestScoreEvent checks that various game events correctly modify the score.
func TestScoreEvent(t *testing.T) {
	mockStorage := &MockScoreStorage{}
	scoring, _ := InitScoring("test", "Test", mockStorage)

	initialScore := scoring.CurrentScore

	// Test wrong letter event
	scoring.ScoreEvent("wrongLetter")
	expectedScore := initialScore - 50
	if scoring.CurrentScore != expectedScore {
		t.Errorf("wrongLetter: expected score %d, got %d", expectedScore, scoring.CurrentScore)
	}
	if scoring.ErrorCount != 1 {
		t.Errorf("wrongLetter: expected error count 1, got %d", scoring.ErrorCount)
	}

	// Test hint event
	scoring.ScoreEvent("hint")
	expectedScore = expectedScore - 100
	if scoring.CurrentScore != expectedScore {
		t.Errorf("hint: expected score %d, got %d", expectedScore, scoring.CurrentScore)
	}
	if scoring.HintCount != 1 {
		t.Errorf("hint: expected hint count 1, got %d", scoring.HintCount)
	}

	// Test right letter event
	scoring.ScoreEvent("rightLetter")
	expectedScore = expectedScore + 25
	if scoring.CurrentScore != expectedScore {
		t.Errorf("rightLetter: expected score %d, got %d", expectedScore, scoring.CurrentScore)
	}
}

// TestGetNScoreEntries_IncludesCurrent verifies that GetNScoreEntries returns
// a combined list of historical scores and the current session's score, sorted correctly.
func TestGetNScoreEntries_IncludesCurrent(t *testing.T) {
	secret := "test text"
	hash := calculateHash(secret)

	mockStorage := &MockScoreStorage{
		Entries: []ScoreHistoryEntry{
			{Hash: hash, Score: 100, Title: "Low"},
			{Hash: hash, Score: 300, Title: "High"},
		},
	}

	scoring, _ := InitScoring(secret, "Test", mockStorage)

	// Set current score to something in between
	scoring.CurrentScore = 200
	if scoring.history.CurrentScore != nil {
		scoring.history.CurrentScore.Score = 200
	}

	// Request top 5 entries (should get all 3)
	entries := scoring.GetNScoreEntries(5)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Expected order: 300 (High), 200 (Current), 100 (Low)
	if entries[0].Score != 300 {
		t.Errorf("expected first entry score 300, got %d", entries[0].Score)
	}
	if entries[1].Score != 200 {
		t.Errorf("expected second entry score 200, got %d", entries[1].Score)
	}
	if entries[2].Score != 100 {
		t.Errorf("expected third entry score 100, got %d", entries[2].Score)
	}
}

// TestGetNumPrevious verifies that GetNumPrevious returns only the count of historical entries.
func TestGetNumPrevious(t *testing.T) {
	secret := "test text"
	hash := calculateHash(secret)

	mockStorage := &MockScoreStorage{
		Entries: []ScoreHistoryEntry{
			{Hash: hash, Score: 10},
			{Hash: hash, Score: 20},
			{Hash: hash, Score: 30},
		},
	}

	scoring, _ := InitScoring(secret, "Test", mockStorage)

	// Current score exists but should not affect the count of *previous* attempts
	scoring.CurrentScore = 50

	count := scoring.GetNumPrevious()
	if count != 3 {
		t.Errorf("expected 3 previous entries, got %d", count)
	}
}
