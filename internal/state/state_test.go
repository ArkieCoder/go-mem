package state

import (
	"context"
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

func TestState_TabJump(t *testing.T) {
	// Secret: "A B C"
	// Mask:   "_ _ _"
	ta := textarea.New()
	sc, _ := scoring.InitScoring("A B C", "Title", &MockStorage{})
	s := NewState("A B C", 20, ta, *sc, GameOptions{})
	s.InitMask()

	// Reveal "A" manually for test setup
	s.Mask[0] = 'A'
	// Pos is 0.

	// Init FSM
	s.FSM.Event(context.Background(), "initGame")

	// Trigger Tab (Jump) via input "tab"
	s.FSM.Event(context.Background(), "input", "tab")

	// Expect jump to next '_'.
	// "A" (0) -> " " (1, ignored/skipped?)
	// Wait, Jump logic finds next '_'.
	// " " is not '_'.
	// "B" (2) is '_'.
	// So Pos should be 2.
	if s.Pos != 2 {
		t.Errorf("Expected Pos to jump to 2, got %d", s.Pos)
	}
}

func TestState_TypeThrough_NoPenalty(t *testing.T) {
	// Secret: "AB"
	// Mask: "AB" (Fully revealed)
	ta := textarea.New()
	sc, _ := scoring.InitScoring("AB", "Title", &MockStorage{})
	s := NewState("AB", 20, ta, *sc, GameOptions{})
	s.InitMask()
	s.Mask[0] = 'A'
	s.Mask[1] = 'B'

	s.FSM.Event(context.Background(), "initGame")

	initialScore := s.Score.CurrentScore

	// Type Wrong Letter on Revealed Character (Pos 0)
	s.FSM.Event(context.Background(), "input", "Z")

	// Should be WrongLetter = false (we move on)
	if s.WrongLetter {
		t.Error("Expected WrongLetter to be false")
	}
	if !s.RevealedCharMistakes[0] {
		t.Error("Expected RevealedCharMistakes[0] to be true")
	}

	// Score should NOT decrease because mask[0] was not '_'
	if s.Score.CurrentScore != initialScore {
		t.Errorf("Expected score %d, got %d", initialScore, s.Score.CurrentScore)
	}

	// Should advance to 1 ('B')
	if s.Pos != 1 {
		t.Errorf("Expected Pos to advance to 1 after wrong input on revealed char, got %d", s.Pos)
	}

	// Type Correct Letter 'B' at Pos 1
	s.FSM.Event(context.Background(), "input", "B")

	// Should Win (Pos stays at 1 because game ended)
	if !s.Win {
		t.Error("Expected Win")
	}
	// Score should increase (Win bonus)
	if s.Score.CurrentScore <= initialScore {
		t.Errorf("Expected score increase, got %d", s.Score.CurrentScore)
	}
}

// MockScoreStorage copy for state tests
type MockStorage struct{}

func (m *MockStorage) LoadAll() ([]scoring.ScoreHistoryEntry, error)     { return nil, nil }
func (m *MockStorage) SaveAll(entries []scoring.ScoreHistoryEntry) error { return nil }

func TestState_SpaceSkipLogic(t *testing.T) {
	// Reproduce/Verify logic for skipping spaces immediately
	ta := textarea.New()
	store := &MockStorage{} // Use existing MockStorage
	sc, _ := scoring.InitScoring("I have", "Title", store)

	opts := GameOptions{FirstLetter: true}

	s := NewState("I have", 20, ta, *sc, opts)
	s.InitMask()
	s.ApplyGameModes(opts)

	s.FSM.Event(context.Background(), "initGame")

	// 1. User types 'I'. Correct.
	s.FSM.Event(context.Background(), "input", "I")

	// Eager skip check: Pos should be 2 ('h') immediately.
	if s.Pos != 2 {
		t.Errorf("After 'I', expected Pos 2 ('h'), got %d", s.Pos)
	}

	// 2. User types 'x' (wrong).
	s.FSM.Event(context.Background(), "input", "x")

	// With "Move On" logic, we expect Pos to advance past 'h' (2 -> 3)
	// 'h' was revealed (First Letter mode).
	if s.Pos != 3 {
		t.Errorf("After 'x', expected Pos 3 ('a'), got %d", s.Pos)
	}
	if s.WrongLetter {
		t.Errorf("After 'x', expected WrongLetter=false (moved on)")
	}
	if !s.RevealedCharMistakes[2] {
		t.Errorf("Expected RevealedCharMistakes[2] to be true")
	}

	// 3. User types 'a' (correct next letter).
	s.FSM.Event(context.Background(), "input", "a")

	// Should advance to 4 ('v')
	if s.Pos != 4 {
		t.Errorf("After 'a', expected Pos 4 ('v'), got %d", s.Pos)
	}
}
