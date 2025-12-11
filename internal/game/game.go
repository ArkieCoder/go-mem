package game

import (
	"context"
	"go-mem/internal/scoring"
	"go-mem/internal/state"

	"github.com/charmbracelet/bubbles/textarea"
)

// Game encapsulates the core game logic, independent of the UI.
type Game struct {
	State *state.State
}

// NewGame initializes a new game instance.
func NewGame(secretMessage string, cardWidth int, ta textarea.Model, scoring scoring.Scoring, opts state.GameOptions) *Game {
	return &Game{
		State: state.NewState(secretMessage, cardWidth, ta, scoring, opts),
	}
}

// Init initializes the game state.
func (g *Game) Init() {
	g.State.SetBracketedPositions()
	g.State.InitMask()

	// Apply game modes
	g.State.ApplyGameModes(g.State.Options)

	g.State.Textarea.SetValue(string(g.State.Mask))
	// Initialize FSM state
	_ = g.State.FSM.Event(context.Background(), "initGame")
}

// HandleTick processes a timer tick.
func (g *Game) HandleTick() {
	if g.State.Win || g.State.Loss || !g.State.TimerEnabled {
		return
	}
	_ = g.State.FSM.Event(context.Background(), "tick")
}

// HandleKeyPress processes a key press and updates the game state.
func (g *Game) HandleKeyPress(ch string) {
	// If game is already over, exit
	if g.State.Win || g.State.Loss {
		return
	}

	// Delegate processing to the FSM
	// We use background context as we don't need cancellation here
	_ = g.State.FSM.Event(context.Background(), "input", ch)
}
