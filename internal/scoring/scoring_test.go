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
