package scoring

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// Scoring manages the game's scoring logic, including event handling,
// bonuses, and history management.
type Scoring struct {
	// public
	CurrentScore   int
	HintCount      int
	ErrorCount     int
	PotentialScore int
	// private
	storage    ScoreStorage // The interface for loading/saving scores.
	history    ScoreHistory
	scoreTable map[string]int
	textHash   string
}

// InitScoring creates and initializes a new Scoring object.
// It loads the score history for the given text using the provided storage interface.
func InitScoring(secretMessage string, title string, storage ScoreStorage) (*Scoring, error) {
	s := &Scoring{
		scoreTable: getScoreTable(),
		storage:    storage,
		textHash:   calculateHash(secretMessage),
	}
	s.PotentialScore = s.scoreTable["baseScore"] * len(secretMessage)
	s.CurrentScore = 0

	// Load all historical entries from storage.
	allEntries, err := s.storage.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("could not load score history: %w", err)
	}

	// Filter entries for the current text.
	filteredEntries := []ScoreHistoryEntry{}
	for _, entry := range allEntries {
		if entry.Hash == s.textHash {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Sort entries to find the high score.
	sort.Slice(filteredEntries, func(i, j int) bool {
		return filteredEntries[i].Score > filteredEntries[j].Score
	})

	s.history.Entries = filteredEntries
	s.history.Attempts = len(filteredEntries)
	if len(filteredEntries) > 0 {
		s.history.HighScoreEntry = &filteredEntries[0]
	}

	// Initialize the current session's score entry.
	s.history.CurrentScore = &ScoreHistoryEntry{
		Hash:      s.textHash,
		Score:     s.CurrentScore,
		Timestamp: time.Now().Format(time.RFC3339),
		Title:     title,
	}

	return s, nil
}

// ScoreEvent updates the score based on a given game event.
func (s *Scoring) ScoreEvent(event string) {
	switch event {
	case "hint":
		s.HintCount++
	case "wrongLetter":
		s.ErrorCount++
	}
	s.CurrentScore += s.scoreTable[event]

	// Update the current score entry in the history.
	if s.history.CurrentScore != nil {
		s.history.CurrentScore.Score = s.CurrentScore
	}
}

func (s *Scoring) AddTimeBonus(seconds int) {
	bonus := seconds * 10
	s.CurrentScore += bonus
	if s.history.CurrentScore != nil {
		s.history.CurrentScore.Score = s.CurrentScore
	}
}

// SaveEntries persists the score for the completed game.
// It reads all scores, updates the list, and writes it back using the storage interface.
func (s *Scoring) SaveEntries() error {
	if s.history.CurrentScore == nil {
		return nil // Nothing to save.
	}

	allEntries, err := s.storage.LoadAll()
	if err != nil {
		return fmt.Errorf("could not load scores for saving: %w", err)
	}

	// Create a new list of entries, excluding any previous scores for the current text.
	updatedEntries := make([]ScoreHistoryEntry, 0)
	for _, entry := range allEntries {
		if entry.Hash != s.textHash {
			updatedEntries = append(updatedEntries, entry)
		}
	}

	// Add the current session's score and all other historical scores for this text.
	updatedEntries = append(updatedEntries, *s.history.CurrentScore)
	for _, entry := range s.history.Entries {
		// Ensure we don't add the current session twice if it was already in history (edge case).
		if entry.Timestamp != s.history.CurrentScore.Timestamp {
			updatedEntries = append(updatedEntries, entry)
		}
	}

	// Save the complete, updated list back to storage.
	return s.storage.SaveAll(updatedEntries)
}

// Accessor methods for score history, delegating to the history object.
func (s *Scoring) GetHighScore() *ScoreHistoryEntry {
	return s.history.GetHighScoreEntry()
}

func (s *Scoring) GetAttempts() int {
	return s.history.Attempts
}

func (s *Scoring) GotHighScore() bool {
	return s.history.GotHighScore()
}

func (s *Scoring) GetNScoreEntries(n int) []ScoreHistoryEntry {
	return s.history.GetNScoreEntries(n)
}

// calculateHash generates a SHA256 hash for the given text.
func calculateHash(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
}

// getScoreTable returns the predefined values for different scoring events.
func getScoreTable() map[string]int {
	return map[string]int{
		"baseScore":    10,
		"rightLetter":  25,
		"wrongLetter":  -50,
		"hint":         -100,
		"wordBonus":    250,
		"messageBonus": 1000,
	}
}
