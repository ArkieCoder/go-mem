package game

import (
	"fmt"
	"go-mem/internal/scoring"
	"go-mem/internal/state"
	"math/rand"

	"github.com/charmbracelet/bubbles/textarea"
)

type Session struct {
	Cards        []CardData
	CurrentIndex int
	CurrentGame  *Game
	GameOptions  state.GameOptions
	ScoreStorage scoring.ScoreStorage

	// Aggregate State
	TotalScore     int
	TotalTimeLimit int
	TimeRemaining  int

	// Batch State
	IsBatch   bool
	Randomize bool
}

func NewSession(cards []CardData, opts state.GameOptions, storage scoring.ScoreStorage, randomize bool) (*Session, error) {
	s := &Session{
		Cards:        cards,
		GameOptions:  opts,
		ScoreStorage: storage,
		IsBatch:      len(cards) > 1,
		Randomize:    randomize,
	}

	// Randomize if requested AND batch mode
	if s.IsBatch && s.Randomize {
		rand.Shuffle(len(s.Cards), func(i, j int) {
			s.Cards[i], s.Cards[j] = s.Cards[j], s.Cards[i]
		})
	}

	// Calculate Total Time Limit
	if opts.TimerLimit > 0 {
		// Fixed time for the whole batch
		s.TotalTimeLimit = opts.TimerLimit
	} else if opts.TimerLimit == -1 {
		// Auto calculate sum of all cards
		totalLen := 0
		for _, c := range cards {
			totalLen += len(c.Content)
		}
		// Logic: len * 2 seconds. Min 10 * num_cards?
		// Or just simple sum.
		// state.NewState uses max(10, len*2).
		// We should replicate that per card or just use total len?
		// "calculated ... for all the cards".
		// Let's sum the individual auto-calcs to be generous.

		totalTime := 0
		for _, c := range cards {
			// Length / 3 seconds.
			l := len(c.Content) / 3
			if l < 10 {
				l = 10
			}
			totalTime += l
		}
		s.TotalTimeLimit = totalTime
	} else {
		// 0 = Disabled
		s.TotalTimeLimit = 0
	}

	s.TimeRemaining = s.TotalTimeLimit

	// Initialize first game
	if err := s.NextGame(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Session) NextGame() error {
	if s.CurrentIndex >= len(s.Cards) {
		return fmt.Errorf("no more cards")
	}

	card := s.Cards[s.CurrentIndex]

	// Construct options for this specific game
	gameOpts := s.GameOptions
	// If timer is enabled (TotalTimeLimit > 0), we pass the REMAINING time as the limit for this game.
	// But we must handle the case where NewState interprets "limit" as "reset to this limit".
	// My logic in Session: I track TimeRemaining.
	// I pass s.TimeRemaining to NewGame.
	// NewGame -> NewState sets TimeLimit = passed value.
	// So Card 2 starts with TimeLimit = 50 (if 50 remained).
	// This works.
	if s.TotalTimeLimit > 0 {
		gameOpts.TimerLimit = s.TimeRemaining
	} else {
		gameOpts.TimerLimit = 0
	}

	// Title logic
	title := card.Title
	if title == "" {
		title = card.Source
		if card.TotalParts > 1 {
			title = fmt.Sprintf("%s #%d", title, card.PartIndex)
		}
	}

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = len(card.Content)
	ta.Prompt = " " // We render manually, but just in case.

	sc, err := scoring.InitScoring(card.Content, title, s.ScoreStorage)
	if err != nil {
		return err
	}
	// Inherit score? No, Scoring is per card.
	// We aggregate manually.

	g := NewGame(card.Content, 0, ta, *sc, gameOpts)
	// Note: cardWidth is passed as 0 here?
	// main.go calculated cardWidth for styling.
	// Game uses cardWidth for... setting Textarea width in Update?
	// And State uses it.
	// We need to calculate cardWidth for each card!
	// main.go had `longestLineLen`.
	// I should duplicate that helper here or export it.
	// I'll calculate it inline for now.

	cw := longestLineLen(card.Content) + 1
	g.State.CardWidth = cw

	g.Init()

	s.CurrentGame = g
	return nil
}

func (s *Session) Update() {
	// Sync session state from current game
	if s.CurrentGame == nil {
		return
	}

	// Sync Timer
	if s.TotalTimeLimit > 0 {
		// The game's timer ticked down.
		// We update our master TimeRemaining.
		s.TimeRemaining = s.CurrentGame.State.TimeRemaining
	}

	// Check Win
	if s.CurrentGame.State.Win {
		// Add score
		s.TotalScore += s.CurrentGame.State.Score.CurrentScore

		// Advance
		s.CurrentIndex++
		if s.CurrentIndex < len(s.Cards) {
			_ = s.NextGame() // Start next
		} else {
			// Session Complete
			// We can mark a flag or just leave CurrentIndex at end
		}
	}
}

func (s *Session) IsFinished() bool {
	return s.CurrentIndex >= len(s.Cards)
}

func (s *Session) IsSessionLoss() bool {
	if s.CurrentGame != nil && s.CurrentGame.State.Loss {
		return true
	}
	return false
}

// Helper duplicated from main (should be shared utils package really)
func longestLineLen(str string) int {
	max := 0
	// naive split
	for _, line := range splitLines(str) {
		if len(line) > max {
			max = len(line)
		}
	}
	return max
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
