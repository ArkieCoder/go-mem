package game

import (
	"go-mem/internal/scoring"
	"go-mem/internal/state"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

// MockStorage implements scoring.ScoreStorage for testing
type MockStorage struct {
	Entries    []scoring.ScoreHistoryEntry
	SaveCalled bool
}

func (m *MockStorage) LoadAll() ([]scoring.ScoreHistoryEntry, error) {
	return m.Entries, nil
}

func (m *MockStorage) SaveAll(entries []scoring.ScoreHistoryEntry) error {
	m.Entries = entries
	m.SaveCalled = true
	return nil
}

func TestGame_Init(t *testing.T) {
	secret := "Hello World"
	ta := textarea.New()
	store := &MockStorage{}
	sc, err := scoring.InitScoring(secret, "Title", store)
	if err != nil {
		t.Fatalf("Failed to init scoring: %v", err)
	}

	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Initial mask should be "_____ _____" (assuming 5 letters, space, 5 letters)
	// 'H' 'e' 'l' 'l' 'o' -> '_'
	// ' ' -> ' ' (shouldIgnore returns true for space)
	// 'W' 'o' 'r' 'l' 'd' -> '_'
	expectedMask := "_____ _____"
	if string(g.State.Mask) != expectedMask {
		t.Errorf("Init mask mismatch. Expected '%s', got '%s'", expectedMask, string(g.State.Mask))
	}

	if g.State.Textarea.Value() != expectedMask {
		t.Errorf("Textarea value mismatch. Expected '%s', got '%s'", expectedMask, g.State.Textarea.Value())
	}
}

func TestGame_Gameplay_Flow(t *testing.T) {
	secret := "Hi"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()
	// Give enough score to survive a penalty
	g.State.Score.CurrentScore = 1000

	// 1. Start: "__"
	if g.State.Textarea.Value() != "__" {
		t.Fatalf("Initial state wrong: %s", g.State.Textarea.Value())
	}

	// 2. Type 'h' (correct)
	g.HandleKeyPress("h")
	// State should be "H_" (internal logic uses secret's case for display)
	if g.State.Textarea.Value() != "H_" {
		t.Errorf("After 'h', expected 'H_', got '%s'", g.State.Textarea.Value())
	}
	if g.State.Pos != 1 {
		t.Errorf("Pos should be 1, got %d", g.State.Pos)
	}
	if g.State.WrongLetter {
		t.Error("WrongLetter should be false")
	}

	// 3. Type 'z' (incorrect)
	initialScore := g.State.Score.CurrentScore
	g.HandleKeyPress("z")
	// Textarea should NOT change for incorrect letter
	if g.State.Textarea.Value() != "H_" {
		t.Errorf("After wrong 'z', expected 'H_', got '%s'", g.State.Textarea.Value())
	}
	if !g.State.WrongLetter {
		t.Error("WrongLetter should be true")
	}
	if g.State.Score.CurrentScore >= initialScore {
		t.Error("Score should have decreased on error")
	}

	// 4. Type 'i' (correct, finishes game)
	g.HandleKeyPress("i")

	if g.State.Textarea.Value() != "Hi" {
		t.Errorf("After 'i', expected 'Hi', got '%s'", g.State.Textarea.Value())
	}
	if g.State.WrongLetter {
		t.Error("WrongLetter should be false after correction")
	}
	if !g.State.Win {
		t.Error("Game should be won")
	}
	if !store.SaveCalled {
		t.Error("SaveEntries should be called on win")
	}
}

func TestGame_Hints(t *testing.T) {
	secret := "AB"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Give enough score to survive hint penalty
	g.State.Score.CurrentScore = 1000

	initialScore := g.State.Score.CurrentScore

	// Use hint
	g.HandleKeyPress("?")

	// Should reveal first letter 'A'
	if g.State.Textarea.Value() != "A_" {
		t.Errorf("After hint, expected 'A_', got '%s'", g.State.Textarea.Value())
	}

	// Check score penalty
	if g.State.Score.CurrentScore >= initialScore {
		t.Error("Score should decrease after hint")
	}

	if g.State.Win {
		t.Error("Should not win yet")
	}
}

func TestGame_LossCondition(t *testing.T) {
	// Setup a game where we can lose quickly
	secret := "LongEnoughToFail"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Force score to be low to trigger loss quickly
	g.State.Score.CurrentScore = 10

	// Type wrong letters until loss
	// Penalty is 50. 10 - 50 = -40 -> Loss.
	g.HandleKeyPress("z")

	if !g.State.Loss {
		t.Error("Should be loss after score drops below 0")
	}
	if !store.SaveCalled {
		t.Error("Should save score on loss")
	}
}

func TestGame_SpaceSkipping(t *testing.T) {
	secret := "A B"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Initial: "_ _" (Spaces revealed by InitMask)
	if g.State.Textarea.Value() != "_ _" {
		t.Fatalf("Init mismatch: '%s'", g.State.Textarea.Value())
	}

	// Type 'a'
	g.HandleKeyPress("a")

	// Expect "A _"
	// 'a' matches 'A', advances. Space is skipped. Next is 'B'.
	if g.State.Textarea.Value() != "A _" {
		t.Errorf("Expected 'A _', got '%s'", g.State.Textarea.Value())
	}

	// Type 'b'
	g.HandleKeyPress("b")
	if g.State.Textarea.Value() != "A B" {
		t.Errorf("Expected 'A B', got '%s'", g.State.Textarea.Value())
	}
	if !g.State.Win {
		t.Error("Should win")
	}
}

func TestGame_RevealAll(t *testing.T) {
	secret := "Hidden"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Initial check
	if g.State.Textarea.Value() != "______" {
		t.Fatalf("Init mismatch: '%s'", g.State.Textarea.Value())
	}

	// Trigger Reveal All (Ctrl+R)
	g.HandleKeyPress("ctrl+r")

	// Check if mask is full secret
	if g.State.Textarea.Value() != "Hidden" {
		t.Errorf("Expected full reveal 'Hidden', got '%s'", g.State.Textarea.Value())
	}

	// Check if game is lost
	if !g.State.Loss {
		t.Error("Game should be marked as Loss after reveal all")
	}

	// Check if revealed flag is set
	if !g.State.Revealed {
		t.Error("Game should be marked as Revealed after reveal all")
	}

	// Check if saved
	if !store.SaveCalled {
		t.Error("Should save score on loss")
	}
}

func TestGame_Timer(t *testing.T) {
	secret := "Short" // 5 chars
	// Time limit logic: max(10, 5 * 2) = 10 seconds.
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)

	// Enable timer (Auto: -1)
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{TimerLimit: -1})
	g.Init()

	if !g.State.TimerEnabled {
		t.Error("Timer should be enabled")
	}
	if g.State.TimeLimit != 10 {
		t.Errorf("Expected time limit 10, got %d", g.State.TimeLimit)
	}
	if g.State.TimeRemaining != 10 {
		t.Errorf("Expected time remaining 10, got %d", g.State.TimeRemaining)
	}

	// Simulate tick
	g.HandleTick()
	if g.State.TimeRemaining != 9 {
		t.Errorf("Expected time remaining 9, got %d", g.State.TimeRemaining)
	}

	// Simulate expiry
	for i := 0; i < 10; i++ {
		g.HandleTick()
	}
	// Remaining should be -1 or 0 depending on logic, but Loss should be true.
	if !g.State.Loss {
		t.Error("Should be loss after timer expires")
	}

	// Reset for Bonus Test
	sc, _ = scoring.InitScoring(secret, "Title", store)
	g = NewGame(secret, 20, ta, *sc, state.GameOptions{TimerLimit: -1})
	g.Init()

	// Initial score should be 0
	if g.State.Score.CurrentScore != 0 {
		t.Errorf("Expected 0 score, got %d", g.State.Score.CurrentScore)
	}

	// Win immediately
	g.HandleKeyPress("s")
	g.HandleKeyPress("h")
	g.HandleKeyPress("o")
	g.HandleKeyPress("r")
	g.HandleKeyPress("t")

	if !g.State.Win {
		t.Fatal("Should win")
	}

	// Bonus: 10 seconds remaining * 10 points = 100 points
	// Plus standard points: 5 chars * 25 = 125
	// Plus word bonus (if applicable): 1 word * 250 = 250
	// Plus message bonus: 1000
	// Total expected: 125 + 250 + 1000 + 100 = 1475.
	// Wait, word bonus logic: `!IsAtEnd && (Secret[Pos] == ' ')`.
	// For "Short", at 't' (last char), IsAtEnd becomes true after typing.
	// `GotCompletedWord` is checked inside `gotMatch` before incrementing Pos?
	// No, `s.Pos` is index of `t`. `Secret[4]` is `t`. Not space.
	// So word bonus might NOT trigger for single word if no trailing space?
	// Let's check `state.go`:
	// `if s.GotCompletedWord() ...`
	// `GotCompletedWord` checks `s.Secret[s.Pos] == ' '`.
	// `t` != ' '. So no word bonus for the last word unless there is a punctuation?
	// `isPunctuation`. `t` is not.

	// So expected: 125 (chars) + 1000 (message) + 100 (time) = 1225.

	expectedMinScore := 1225
	if g.State.Score.CurrentScore < expectedMinScore {
		t.Errorf("Expected score at least %d, got %d", expectedMinScore, g.State.Score.CurrentScore)
	}

	// Check if time bonus specifically was added?
	// Hard to check exact breakdown without inspecting internals or calculating exact expected.
	// But getting > 1125 implies time bonus was added.
}

func TestGame_TypeThroughRevealed(t *testing.T) {
	secret := "Hello World"
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)

	// Enable First Letter mode
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{FirstLetter: true})
	g.Init()

	// Initial mask should have First Letters revealed
	// "H____ W____"
	expectedMask := "H____ W____"
	if string(g.State.Mask) != expectedMask {
		t.Fatalf("Init mask mismatch. Expected '%s', got '%s'", expectedMask, string(g.State.Mask))
	}

	// At this point, Pos should be at 'e' (index 1).
	// 'H' (0) was revealed and skipped.
	if g.State.Pos != 1 {
		t.Errorf("Pos should be 1 after skipping 'H', got %d", g.State.Pos)
	}

	// User types 'H' (the revealed letter). Should be ignored.
	initialScore := g.State.Score.CurrentScore
	g.HandleKeyPress("h")

	// Pos should still be 1
	if g.State.Pos != 1 {
		t.Errorf("Pos should remain 1 after typing revealed letter, got %d", g.State.Pos)
	}
	// Score should NOT change (not an error)
	if g.State.Score.CurrentScore != initialScore {
		t.Errorf("Score should not change, got %d", g.State.Score.CurrentScore)
	}
	// WrongLetter should be false
	if g.State.WrongLetter {
		t.Error("Should not flag as wrong letter")
	}

	// Now type 'e' (the actual next letter)
	g.HandleKeyPress("e")

	// Should advance
	if g.State.Pos != 2 {
		t.Errorf("Pos should be 2 after typing 'e', got %d", g.State.Pos)
	}
	if string(g.State.Mask) != "He___ W____" {
		t.Errorf("Mask mismatch after 'e'. Got '%s'", string(g.State.Mask))
	}
}

func TestGame_TypeThrough_EdgeCase(t *testing.T) {
	secret := "One two three four."
	ta := textarea.New()
	store := &MockStorage{}
	sc, _ := scoring.InitScoring(secret, "Title", store)

	// Create game with no special options initially
	g := NewGame(secret, 20, ta, *sc, state.GameOptions{})
	g.Init()

	// Manually set the Mask to simulate -nr=3 scenario
	// Secret: One two three four.
	// Mask:   ___ _w_ ____e ___r.

	g.State.Mask[5] = 'w'
	g.State.Mask[11] = 'e'
	g.State.Mask[12] = 'e'
	g.State.Mask[17] = 'r'

	g.State.Textarea.SetValue(string(g.State.Mask))

	// Type "One "
	g.HandleKeyPress("O")
	g.HandleKeyPress("n")
	g.HandleKeyPress("e")
	g.HandleKeyPress(" ") // Space

	// Pos should be 4 ('t')
	if g.State.Pos != 4 {
		t.Fatalf("Pos should be 4 after 'One ', got %d", g.State.Pos)
	}

	// Type 't'
	g.HandleKeyPress("t")

	// Pos stays at 5 ('w') because skipping happens lazily on NEXT input
	if g.State.Pos != 5 {
		t.Fatalf("Pos should be 5 after 't' (waiting at 'w'), got %d", g.State.Pos)
	}

	// Type 'w' (the revealed char)
	// Should be ignored
	initialScore := g.State.Score.CurrentScore
	g.HandleKeyPress("w")

	// Now Pos should be 6 ('o') because SkipRevealed ran
	if g.State.Pos != 6 {
		t.Errorf("Pos should be 6 after 'w' ignored, got %d", g.State.Pos)
	}
	if g.State.WrongLetter {
		t.Errorf("Should not be wrong letter")
	}
	if g.State.Score.CurrentScore != initialScore {
		t.Errorf("Score changed on 'w'")
	}

	// Type 'o'
	// This was the failure point: 'o' matches Secret[6] ('o')
	// BUT 'o' also matches 'O' at Secret[0] (via backward scan for typethrough)
	// Logic should prioritize correct letter
	g.HandleKeyPress("o")

	// Should advance
	// 'o' matched at 6. advance -> 7.
	if g.State.Pos != 7 {
		t.Errorf("Pos should be 7 after 'o', got %d", g.State.Pos)
	}
}
