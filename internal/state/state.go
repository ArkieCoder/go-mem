package state

import (
	"context"
	"go-mem/internal/scoring"
	"math/rand"
	"slices"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/looplab/fsm"
)

// ... GameOptions and State structs remain the same ...
type GameOptions struct {
	TimerLimit  int // -1 auto, 0 off, >0 seconds
	FirstLetter bool
	NRandom     int
	NWords      int
}

type State struct {
	Textarea           textarea.Model
	Mask               []rune
	Secret             []rune
	Pos                int
	Win                bool // To determine if the user has won
	Loss               bool // To determine if the user has lost
	Revealed           bool // To determine if the user revealed the card
	WrongLetter        bool // To determine if the last typed character was wrong
	Score              scoring.Scoring
	CardWidth          int
	BracketedPositions []int
	FSM                *fsm.FSM
	CurrentChar        string // Current character being processed
	TimerEnabled       bool
	TimeLimit          int // Total time in seconds
	TimeRemaining      int // Current time remaining in seconds
	Options            GameOptions
}

// ... NewState ...
func NewState(
	secretMessage string,
	cardWidth int,
	ta textarea.Model,
	scoring scoring.Scoring,
	opts GameOptions,
) *State {
	s := &State{
		Textarea:     ta,
		Secret:       []rune(secretMessage),
		Pos:          0,
		WrongLetter:  false,
		Score:        scoring,
		CardWidth:    cardWidth,
		TimerEnabled: opts.TimerLimit != 0,
		Options:      opts,
	}

	if s.TimerEnabled {
		limit := opts.TimerLimit
		if limit == -1 {
			// Calculate time limit: Length / 3 seconds (approx 180 CPM).
			// At least 10 seconds.
			limit = len(s.Secret) / 3
			if limit < 10 {
				limit = 10
			}
		}
		s.TimeLimit = limit
		s.TimeRemaining = limit
	}

	s.FSM = fsm.NewFSM(
		"start",
		getStateTransitions(),
		getStateCallbacks(s),
	)

	return s
}

// ... ApplyGameModes, Reveal methods ...
func (s *State) ApplyGameModes(opts GameOptions) {
	if opts.FirstLetter {
		s.RevealFirstLetters()
	}
	if opts.NRandom > 0 {
		s.RevealRandomLetters(opts.NRandom)
	}
	if opts.NWords > 0 {
		s.RevealRandomWords(opts.NWords)
	}
	// After applying modes, ensure we are advanced past any initially revealed characters
	s.SkipRevealed()
}

func (s *State) RevealFirstLetters() {
	inWord := false
	for i, ch := range s.Secret {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			if !inWord {
				// Start of word
				s.Mask[i] = ch
				inWord = true
			}
		} else {
			inWord = false
		}
	}
}

func (s *State) RevealRandomLetters(n int) {
	// Find all unrevealed letter indices
	candidates := []int{}
	for i, ch := range s.Secret {
		if s.Mask[i] == '_' && (unicode.IsLetter(ch) || unicode.IsDigit(ch)) {
			candidates = append(candidates, i)
		}
	}

	// Shuffle and pick n
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	count := n
	if count > len(candidates) {
		count = len(candidates)
	}

	for i := 0; i < count; i++ {
		idx := candidates[i]
		s.Mask[idx] = s.Secret[idx]
	}
}

func (s *State) RevealRandomWords(n int) {
	type wordSpan struct {
		start, end int
	}
	var words []wordSpan
	inWord := false
	start := 0

	for i, ch := range s.Secret {
		isAlphanum := unicode.IsLetter(ch) || unicode.IsDigit(ch)
		if isAlphanum {
			if !inWord {
				start = i
				inWord = true
			}
		} else {
			if inWord {
				words = append(words, wordSpan{start, i})
				inWord = false
			}
		}
	}
	// Check last word
	if inWord {
		words = append(words, wordSpan{start, len(s.Secret)})
	}

	rand.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})

	count := n
	if count > len(words) {
		count = len(words)
	}

	for i := 0; i < count; i++ {
		span := words[i]
		for j := span.start; j < span.end; j++ {
			s.Mask[j] = s.Secret[j]
		}
	}
}

func (s *State) SkipRevealed() {
	// Skip spaces and punctuation AND ALREADY REVEALED letters in the secret message
	for s.Pos < len(s.Secret) && (s.ShouldIgnore(string(s.Secret[s.Pos])) || slices.Contains(s.BracketedPositions, s.Pos) || s.Mask[s.Pos] != '_') {
		// Only reveal if it's punctuation/brackets. If it's already revealed (Mask != '_'), we just skip.
		// But we need to ensure Mask is consistent.
		if s.Mask[s.Pos] == '_' {
			s.Mask[s.Pos] = s.Secret[s.Pos]
		}
		s.Pos++
	}
}

// ... getStateTransitions ...
func getStateTransitions() []fsm.EventDesc {
	return fsm.Events{
		{Name: "initGame", Src: []string{"start"}, Dst: "idle"},
		{Name: "input", Src: []string{"idle"}, Dst: "checkGameState"},

		// Game State Checking
		{Name: "gameEnd", Src: []string{"checkGameState", "evaluating", "revealingAll"}, Dst: "endState"},
		{Name: "proceed", Src: []string{"checkGameState"}, Dst: "processChar"},
		{Name: "revealAll", Src: []string{"checkGameState"}, Dst: "revealingAll"},

		// Character Processing
		{Name: "ignore", Src: []string{"processChar"}, Dst: "evaluating"},
		{Name: "reveal", Src: []string{"processChar"}, Dst: "revealNextChar"},
		{Name: "check", Src: []string{"processChar"}, Dst: "checkCorrectness"},

		// Actions
		{Name: "revealed", Src: []string{"revealNextChar"}, Dst: "updateMask"},
		{Name: "match", Src: []string{"checkCorrectness"}, Dst: "gotMatch"},
		{Name: "mismatch", Src: []string{"checkCorrectness"}, Dst: "noMatch"},

		{Name: "matched", Src: []string{"gotMatch"}, Dst: "updateMask"},
		{Name: "gameEnd", Src: []string{"gotMatch"}, Dst: "endState"}, // Allow early exit from gotMatch
		{Name: "notMatched", Src: []string{"noMatch"}, Dst: "updateScore"},

		{Name: "advance", Src: []string{"updateMask"}, Dst: "advancing"},

		{Name: "advanced", Src: []string{"advancing"}, Dst: "evaluating"},
		{Name: "scoreCalculated", Src: []string{"updateScore"}, Dst: "evaluating"},

		// End Loop
		{Name: "wait", Src: []string{"evaluating"}, Dst: "idle"},
		{Name: "tick", Src: []string{"idle"}, Dst: "timeCheck"},
		{Name: "timePassed", Src: []string{"timeCheck"}, Dst: "idle"},
		{Name: "timeExpired", Src: []string{"timeCheck"}, Dst: "endState"},
	}
}

// ... getStateCallbacks ...
func getStateCallbacks(s *State) map[string]fsm.Callback {
	return fsm.Callbacks{
		"enter_timeCheck": func(ctx context.Context, e *fsm.Event) {
			s.TimeRemaining--
			if s.TimeRemaining <= 0 {
				s.Loss = true
				e.FSM.Event(ctx, "timeExpired")
				return
			}
			e.FSM.Event(ctx, "timePassed")
		},
		"enter_checkGameState": func(ctx context.Context, e *fsm.Event) {
			// Capture the input character
			if len(e.Args) > 0 {
				s.CurrentChar = e.Args[0].(string)
			} else {
				s.CurrentChar = ""
			}

			// Check if the game is already won
			if s.GotCorrectMessage() {
				s.Win = true
				e.FSM.Event(ctx, "gameEnd")
				return
			}

			// Check for exit request
			if IsExitRequested(s.CurrentChar) {
				s.Loss = true
				e.FSM.Event(ctx, "gameEnd")
				return
			}

			// Check if previous move caused loss (e.g. score drop)
			if s.Score.CurrentScore < 0 {
				s.Loss = true
				e.FSM.Event(ctx, "gameEnd")
				return
			}

			// Check for reveal request
			if IsRevealRequested(s.CurrentChar) {
				e.FSM.Event(ctx, "revealAll")
				return
			}

			e.FSM.Event(ctx, "proceed")
		},
		"enter_revealingAll": func(ctx context.Context, e *fsm.Event) {
			s.Mask = make([]rune, len(s.Secret))
			copy(s.Mask, s.Secret)
			s.Textarea.SetValue(string(s.Mask))
			s.Loss = true // User gave up
			s.Revealed = true
			e.FSM.Event(ctx, "gameEnd")
		},
		"enter_processChar": func(ctx context.Context, e *fsm.Event) {
			// Use the helper to skip revealed logic (ensures Pos is at first UNREVEALED char)
			s.SkipRevealed()

			// Update UI to show skipped chars immediately
			s.Textarea.SetValue(string(s.Mask))

			// Check if we reached end after skipping
			if s.Pos >= len(s.Secret) {
				if string(s.Mask) == string(s.Secret) {
					s.Win = true
					s.Score.ScoreEvent("messageBonus") // Bonus logic
					if s.TimerEnabled {
						s.Score.AddTimeBonus(s.TimeRemaining)
					}
					e.FSM.Event(ctx, "gameEnd")
					return
				}
				// If not win (weird state?), ignore.
				e.FSM.Event(ctx, "ignore")
				return
			}

			// PRIORITY: If the user typed the CORRECT next letter, accept it!
			// This prevents mistakenly ignoring a character because it appeared previously.
			if s.IsCorrectLetter(s.CurrentChar) {
				e.FSM.Event(ctx, "check")
				return
			}

			// Check if user typed a character that is ALREADY REVEALED immediately before Pos
			// Scan backwards from Pos-1 to find the contiguous block of revealed characters
			for i := s.Pos - 1; i >= 0; i-- {
				// If we hit an unrevealed char (shouldn't happen if Pos is correct, but safe check) or a gap?
				// Wait, SkipRevealed skips spaces too.
				// Spaces are in Mask as ' '.
				// So Mask[i] != '_' covers letters AND spaces/punctuation.
				if s.Mask[i] == '_' {
					break // End of the revealed block (going backwards)
				}

				// Stop scanning if we hit a word boundary (space or punctuation)
				// This prevents matching letters from previous words in the same line
				if s.ShouldIgnore(string(s.Secret[i])) {
					break
				}

				charStr := string(s.Secret[i])
				// Case-insensitive check
				if strings.EqualFold(charStr, s.CurrentChar) {
					// User typed a character that is in the revealed block immediately preceding the current position.
					// Assume they are "typing through" the revealed text.
					e.FSM.Event(ctx, "ignore")
					return
				}
			}

			// Check if we should ignore this user input (e.g. space typed by user)
			if s.ShouldIgnore(s.CurrentChar) {
				e.FSM.Event(ctx, "ignore")
				return
			}

			// Check for hint request
			if s.CurrentChar == "?" {
				e.FSM.Event(ctx, "reveal")
				return
			}

			// Normal check
			e.FSM.Event(ctx, "check")
		},
		"enter_checkCorrectness": func(ctx context.Context, e *fsm.Event) {
			// If we are in error state (WrongLetter), only accept correct letter
			if s.WrongLetter {
				if s.IsCorrectLetter(s.CurrentChar) {
					s.WrongLetter = false // Clear error state
					e.FSM.Event(ctx, "match")
				} else {
					// Still wrong
					e.FSM.Event(ctx, "mismatch")
				}
			} else {
				// Normal state
				if s.IsCorrectLetter(s.CurrentChar) {
					e.FSM.Event(ctx, "match")
				} else if s.IsIncorrectLetter(s.CurrentChar) {
					e.FSM.Event(ctx, "mismatch")
				} else {
					// Should not happen given logic, but treat as ignore?
					// If we are at end of string?
					e.FSM.Event(ctx, "ignore")
				}
			}
		},
		"enter_gotMatch": func(ctx context.Context, e *fsm.Event) {
			s.Mask[s.Pos] = s.Secret[s.Pos]
			s.Score.ScoreEvent("rightLetter")

			// Check word completion BEFORE we advance Pos
			// (GotCompletedWord checks s.Secret[s.Pos] which is current char)
			if s.GotCompletedWord() {
				s.Score.ScoreEvent("wordBonus")
			}

			// If the message is complete, win immediately
			if string(s.Mask) == string(s.Secret) {
				s.Win = true
				s.Score.ScoreEvent("messageBonus") // Apply bonus here as it won't be applied in evaluating
				if s.TimerEnabled {
					s.Score.AddTimeBonus(s.TimeRemaining)
				}
				s.Textarea.SetValue(string(s.Mask)) // Update UI one last time before ending
				e.FSM.Event(ctx, "gameEnd")         // Skip updateMask/advance, go straight to end
				return
			}

			e.FSM.Event(ctx, "matched")
		},
		"enter_noMatch": func(ctx context.Context, e *fsm.Event) {
			s.WrongLetter = true
			s.Score.ScoreEvent("wrongLetter")
			e.FSM.Event(ctx, "notMatched")
		},
		"enter_revealNextChar": func(ctx context.Context, e *fsm.Event) {
			// Hint logic: Find next hidden char
			tempPos := s.Pos
			for tempPos < len(s.Secret) && (s.ShouldIgnore(string(s.Secret[tempPos])) || s.Mask[tempPos] != '_') {
				tempPos++
			}

			if tempPos < len(s.Secret) && s.Mask[tempPos] == '_' {
				s.Mask[tempPos] = s.Secret[tempPos]
				s.Score.ScoreEvent("hint")
			}

			e.FSM.Event(ctx, "revealed")
		},
		"enter_updateMask": func(ctx context.Context, e *fsm.Event) {
			s.Textarea.SetValue(string(s.Mask))
			e.FSM.Event(ctx, "advance")
		},
		"enter_advancing": func(ctx context.Context, e *fsm.Event) {
			s.Pos++
			e.FSM.Event(ctx, "advanced")
		},
		"enter_updateScore": func(ctx context.Context, e *fsm.Event) {
			// Score updated in previous events, just transition
			// Force update textarea (to show red cursor if implemented in view based on WrongLetter)
			s.Textarea.SetValue(string(s.Mask))
			e.FSM.Event(ctx, "scoreCalculated")
		},
		"enter_evaluating": func(ctx context.Context, e *fsm.Event) {
			if s.IsGameOver() {
				if s.LostGame() {
					s.Loss = true
				} else {
					s.Win = true
					s.Score.ScoreEvent("messageBonus")
				}
				e.FSM.Event(ctx, "gameEnd")
				return
			}

			// Also check score again (redundant but safe)
			if s.Score.CurrentScore < 0 {
				s.Loss = true
				e.FSM.Event(ctx, "gameEnd")
				return
			}

			e.FSM.Event(ctx, "wait")
		},
		"enter_endState": func(ctx context.Context, e *fsm.Event) {
			s.Score.SaveEntries()
		},
	}
}
