package scoring

import (
	"sort"
)

// ScoreHistory holds the score data for a particular text, including
// past entries and the current session's score.
type ScoreHistory struct {
	Entries        []ScoreHistoryEntry
	HighScoreEntry *ScoreHistoryEntry
	CurrentScore   *ScoreHistoryEntry
	Attempts       int
}

// ScoreHistoryEntry represents a single score record for a given text.
type ScoreHistoryEntry struct {
	Hash      string `json:"hash"`
	Score     int    `json:"score"`
	Timestamp string `json:"timestamp"`
	Title     string `json:"title"`
}

// GetHighScoreEntry returns the highest score entry from the loaded history.
func (sh ScoreHistory) GetHighScoreEntry() *ScoreHistoryEntry {
	return sh.HighScoreEntry
}

// GetNScoreEntries returns the top N score entries from the history, sorted by score.
func (sh ScoreHistory) GetNScoreEntries(n int) []ScoreHistoryEntry {
	// Make a copy to avoid modifying the original slice.
	entriesCopy := make([]ScoreHistoryEntry, len(sh.Entries))
	copy(entriesCopy, sh.Entries)

	sort.Slice(entriesCopy, func(i, j int) bool {
		return entriesCopy[i].Score > entriesCopy[j].Score
	})

	if len(entriesCopy) < n {
		return entriesCopy
	}
	return entriesCopy[:n]
}

// GotHighScore checks if the current score is greater than or equal to the
// previously recorded high score.
func (sh ScoreHistory) GotHighScore() bool {
	if sh.HighScoreEntry == nil || sh.CurrentScore == nil {
		// If there's no high score or no current score, it's vacuously a "high score".
		return true
	}
	return sh.CurrentScore.Score >= sh.HighScoreEntry.Score
}
