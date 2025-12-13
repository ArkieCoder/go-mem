package game

import (
	"go-mem/internal/state"
	"testing"
)

func TestSession_Init(t *testing.T) {
	cards := []CardData{
		{Content: "Card1", Source: "src1"},
		{Content: "Card2", Source: "src2"},
	}
	opts := state.GameOptions{TimerLimit: -1} // Auto
	store := &MockStorage{}

	sess, err := NewSession(cards, opts, store, false)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if !sess.IsBatch {
		t.Error("Should be batch mode")
	}
	if sess.TotalTimeLimit == 0 {
		t.Error("TotalTimeLimit should be calculated")
	}
	// Card1 (5 chars) -> 10s. Card2 (5 chars) -> 10s. Total 20s.
	if sess.TotalTimeLimit != 20 {
		t.Errorf("Expected 20s total time, got %d", sess.TotalTimeLimit)
	}
	if sess.TimeRemaining != 20 {
		t.Errorf("Expected 20s remaining, got %d", sess.TimeRemaining)
	}

	if sess.CurrentGame == nil {
		t.Error("CurrentGame should be initialized")
	}
	if string(sess.CurrentGame.State.Secret) != "Card1" {
		t.Errorf("First game should be Card1, got %s", string(sess.CurrentGame.State.Secret))
	}
}

func TestSession_Progression(t *testing.T) {
	cards := []CardData{
		{Content: "A", Source: "src1"},
		{Content: "B", Source: "src2"},
	}
	opts := state.GameOptions{TimerLimit: 0} // No timer
	store := &MockStorage{}

	sess, _ := NewSession(cards, opts, store, false)

	// Win Game 1
	sess.CurrentGame.HandleKeyPress("A") // Win
	// Check Win
	if !sess.CurrentGame.State.Win {
		t.Fatal("Game 1 should be won")
	}

	// Update Session
	sess.Update()

	// Manually advance (simulating main loop)
	sess.CurrentIndex++
	_ = sess.NextGame()

	// Should have moved to Game 2
	if sess.CurrentIndex != 1 {
		t.Errorf("Should be at index 1, got %d", sess.CurrentIndex)
	}
	if string(sess.CurrentGame.State.Secret) != "B" {
		t.Errorf("Current game should be B, got %s", string(sess.CurrentGame.State.Secret))
	}

	// Win Game 2
	sess.CurrentGame.HandleKeyPress("B")
	sess.Update()

	// Manually advance
	sess.CurrentIndex++

	// Should be finished
	if !sess.IsFinished() {
		t.Error("Session should be finished")
	}

	// Check score aggregation
	// Each game: 25 pts (char) + 1000 pts (message) = 1025.
	// Total: 2050.
	if sess.TotalScore != 2050 {
		t.Errorf("Expected total score 2050, got %d", sess.TotalScore)
	}
}

func TestSession_TimePersistence(t *testing.T) {
	cards := []CardData{
		{Content: "A", Source: "src1"},
		{Content: "B", Source: "src2"},
	}
	// Fixed timer 100s
	opts := state.GameOptions{TimerLimit: 100}
	store := &MockStorage{}

	sess, _ := NewSession(cards, opts, store, false)

	// Simulate 10s passing in Game 1
	// We manually decrement Game 1 state?
	// HandleTick calls event.
	// Or we manually set TimeRemaining for test.
	sess.CurrentGame.State.TimeRemaining = 90

	// Win Game 1
	sess.CurrentGame.HandleKeyPress("A")
	sess.Update()

	// Manually advance
	sess.CurrentIndex++
	_ = sess.NextGame()

	// Session TimeRemaining should be 90
	if sess.TimeRemaining != 90 {
		t.Errorf("Session time should be 90, got %d", sess.TimeRemaining)
	}

	// Game 2 should start with 90s limit
	if sess.CurrentGame.State.TimeLimit != 90 {
		t.Errorf("Game 2 limit should be 90, got %d", sess.CurrentGame.State.TimeLimit)
	}
}
