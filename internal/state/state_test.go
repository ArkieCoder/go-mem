package state

import (
	"go-mem/internal/scoring"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

func TestState_SetBracketedPositions(t *testing.T) {
	// Case 1: Brackets containing text
	secret := "Hello [World]!"
	s := NewState(secret, 20, textarea.New(), scoring.Scoring{}, GameOptions{})
	s.SetBracketedPositions()

	expectedSecret := "Hello World!"
	if string(s.Secret) != expectedSecret {
		t.Errorf("Expected secret '%s', got '%s'", expectedSecret, string(s.Secret))
	}

	expectedPos := []int{6, 7, 8, 9, 10}
	if len(s.BracketedPositions) != len(expectedPos) {
		t.Fatalf("Expected %d bracketed positions, got %d", len(expectedPos), len(s.BracketedPositions))
	}

	for i, pos := range expectedPos {
		if s.BracketedPositions[i] != pos {
			t.Errorf("Position mismatch at index %d: expected %d, got %d", i, pos, s.BracketedPositions[i])
		}
	}
}

func TestState_SetBracketedPositions_Multiline(t *testing.T) {
	// Case 2: Brackets across lines
	secret := "Hello [World\nAgain]!"
	// "Hello " (6)
	// [ starts.
	// W(6) o(7) r(8) l(9) d(10) \n(11) A(12) g(13) a(14) i(15) n(16)
	// ] ends.
	// ! (17)

	s := NewState(secret, 20, textarea.New(), scoring.Scoring{}, GameOptions{})
	s.SetBracketedPositions()

	expectedSecret := "Hello World\nAgain!"
	if string(s.Secret) != expectedSecret {
		t.Errorf("Expected secret '%s', got '%s'", expectedSecret, string(s.Secret))
	}

	// Expect 11 chars in bracket (World\nAgain) -> 5 + 1 + 5 = 11
	if len(s.BracketedPositions) != 11 {
		t.Errorf("Expected 11 bracketed positions, got %d", len(s.BracketedPositions))
	}
}

func TestState_InitMask(t *testing.T) {
	ta := textarea.New()
	s := NewState("A B", 20, ta, scoring.Scoring{}, GameOptions{})
	s.InitMask()

	expectedMask := "_ _"
	if string(s.Mask) != expectedMask {
		t.Errorf("Expected mask '%s', got '%s'", expectedMask, string(s.Mask))
	}

	// Test with Brackets
	s = NewState("A [B]", 20, ta, scoring.Scoring{}, GameOptions{})
	s.SetBracketedPositions() // Secret becomes "A B", B is bracketed
	s.InitMask()

	expectedMask2 := "_ B"
	if string(s.Mask) != expectedMask2 {
		t.Errorf("Expected mask '%s', got '%s'", expectedMask2, string(s.Mask))
	}
}

func TestState_ShouldIgnore(t *testing.T) {
	s := State{}

	tests := []struct {
		input  string
		expect bool
	}{
		{" ", true},
		{".", true},
		{"!", true},
		{"?", false},
		{"a", false},
		{"A", false},
		{"1", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := s.ShouldIgnore(tt.input); got != tt.expect {
			t.Errorf("ShouldIgnore('%s') = %v, expected %v", tt.input, got, tt.expect)
		}
	}
}

func TestState_IsCorrectLetter(t *testing.T) {
	s := NewState("Hello", 20, textarea.New(), scoring.Scoring{}, GameOptions{})
	s.Pos = 0

	if !s.IsCorrectLetter("h") {
		t.Error("Expected 'h' to be correct for 'H'")
	}
	if !s.IsCorrectLetter("H") {
		t.Error("Expected 'H' to be correct for 'H'")
	}
	if s.IsCorrectLetter("e") {
		t.Error("Expected 'e' to be incorrect for 'H'")
	}

	s.Pos = 4 // 'o'
	if !s.IsCorrectLetter("o") {
		t.Error("Expected 'o' to be correct")
	}

	s.Pos = 5 // End
	if s.IsCorrectLetter("anything") {
		t.Error("Should return false when at end")
	}
}

func TestState_GotCompletedWord(t *testing.T) {
	s := NewState("Hi World", 20, textarea.New(), scoring.Scoring{}, GameOptions{})
	s.Pos = 0

	if s.GotCompletedWord() {
		t.Error("Should not be completed word at start")
	}

	s.Pos = 2
	if !s.GotCompletedWord() {
		t.Error("Should be completed word at space")
	}
}

func TestState_WinLoss(t *testing.T) {
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring("A", "Title", store)
	s := NewState("A", 20, ta, *sc, GameOptions{})

	s.Score.CurrentScore = -1
	if !s.IsGameOver() {
		t.Error("Should be game over if score < 0")
	}

	if !s.LostGame() {
		t.Error("Should be lost game if score < 0")
	}
}

// MockScoreStorage copy for state tests
type MockStorage struct{}

func (m *MockStorage) LoadAll() ([]scoring.ScoreHistoryEntry, error)     { return nil, nil }
func (m *MockStorage) SaveAll(entries []scoring.ScoreHistoryEntry) error { return nil }
