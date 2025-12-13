package main

import (
	"flag"
	"fmt"

	"go-mem/internal/game"
	"go-mem/internal/scoring"
	"go-mem/internal/state"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red for incorrect inputs
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green for correct input
	scoreStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Color for the score
	boldStyle   = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
)

type LocalState struct {
	Session       *game.Session
	QuitNextCycle bool
	Quitting      bool
}

type TickMsg time.Time
type QuitMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func initialModel(paths []string, opts state.GameOptions, randomize bool) (*LocalState, error) {
	cards, err := game.LoadCards(paths)
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		return nil, fmt.Errorf("no cards found in provided paths")
	}

	// Create the concrete storage implementation.
	storage, err := scoring.NewJSONFileStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create score storage: %w", err)
	}

	// Session handles scoring init per game.

	sess, err := game.NewSession(cards, opts, storage, randomize)
	if err != nil {
		return nil, err
	}

	return &LocalState{
		Session: sess,
	}, nil
}

func noOp() tea.Msg {
	return nil
}

func (s *LocalState) Init() tea.Cmd {
	// Session initializes first game automatically
	if s.Session.CurrentGame.State.TimerEnabled {
		return tickCmd()
	}
	return noOp
}

func (s *LocalState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	currentGame := s.Session.CurrentGame

	switch msg := msg.(type) {
	case QuitMsg:
		return s, tea.Quit
	case TickMsg:
		if s.Quitting {
			return s, func() tea.Msg { return QuitMsg{} }
		}
		currentGame.HandleTick()
		s.Session.Update() // Check for session loss or transition
		if s.Session.IsSessionLoss() || s.Session.IsFinished() || currentGame.State.Win {
			s.Quitting = true
			return s, func() tea.Msg { return QuitMsg{} }
		}
		return s, tickCmd()
	case tea.WindowSizeMsg:
		// Resize logic should apply to current game
		currentGame.State.Textarea.SetWidth(currentGame.State.CardWidth + 1)
		lineCount := len(strings.Split(string(currentGame.State.Secret), "\n"))
		currentGame.State.Textarea.SetHeight(lineCount)
	case tea.KeyMsg:
		ch := msg.String()

		// Handle exit request
		if state.IsExitRequested(ch) {
			return s, tea.Quit
		}

		// Check if game over before processing?
		if currentGame.State.Win || currentGame.State.Loss {
			// If already over, maybe we are waiting to quit?
			// Session update should have handled transitions.
			// If we are here, maybe we are at the end of session?
			// Note: With the loop refactor, we quit on any Win/Loss of the CURRENT game.
			if s.Session.IsFinished() || s.Session.IsSessionLoss() || currentGame.State.Win {
				s.Quitting = true
				return s, func() tea.Msg { return QuitMsg{} }
			}
		}

		currentGame.HandleKeyPress(ch)
		s.Session.Update() // Check transitions

		// Note: With loop refactor, we check for single game win too
		if s.Session.IsSessionLoss() || s.Session.IsFinished() || currentGame.State.Win {
			s.Quitting = true
			return s, func() tea.Msg { return QuitMsg{} }
		}

		// If Session Update switched games (NextGame), View will handle rendering new game state.
	}

	return s, nil
}

func (s *LocalState) RenderBoard() string {
	var b strings.Builder
	// Render board for CURRENT game
	g := s.Session.CurrentGame
	mask := g.State.Mask
	pos := g.State.Pos
	bracketed := g.State.BracketedPositions

	for i, r := range mask {
		style := lipgloss.NewStyle()

		// Apply placeholder style (bold)
		if slices.Contains(bracketed, i) {
			style = style.Bold(true)
		}

		// Apply cursor style
		if !g.State.Win && !g.State.Loss && i == pos {
			if g.State.WrongLetter {
				// If character is already revealed (visible), use Red Underline
				if mask[i] != '_' {
					style = style.Foreground(lipgloss.Color("9")).Underline(true)
				} else {
					// Red Block Cursor for hidden char
					style = style.Background(lipgloss.Color("9"))
				}
			} else {
				// Reverse video for normal cursor
				style = style.Reverse(true)
			}
		}

		b.WriteString(style.Render(string(r)))
	}
	return b.String()
}

func (s *LocalState) View() string {
	g := s.Session.CurrentGame

	// Determine which card to display.
	// If session is finished, show the last card (the one just completed).
	var card game.CardData
	if s.Session.IsFinished() {
		if len(s.Session.Cards) > 0 {
			card = s.Session.Cards[len(s.Session.Cards)-1]
		}
	} else {
		card = s.Session.Cards[s.Session.CurrentIndex]
	}

	// 1. Render Banner
	secretMessageStr := string(g.State.Secret)

	smLongestLineLen := longestLineLen(secretMessageStr)

	var textTitle string
	if card.Title != "" {
		textTitle = card.Title
	} else {
		fileExt := filepath.Ext(card.Source)
		textTitle = titleCaseToTitle(filepath.Base(strings.TrimSuffix(card.Source, fileExt)))
		if card.TotalParts > 1 {
			textTitle = fmt.Sprintf("%s #%d", textTitle, card.PartIndex)
		}
	}

	bannerTxt := fmt.Sprintf("┃ CARD: %s | LOC: %s", textTitle, card.Source)

	cardWidth := smLongestLineLen + 1
	if len(bannerTxt) > cardWidth {
		cardWidth = len(bannerTxt) + 1
	}

	// Ensure banner padding matches
	paddingNeeded := cardWidth - len(bannerTxt) + 4
	if paddingNeeded < 0 {
		paddingNeeded = 0
	}
	bannerTxt += strings.Repeat(" ", paddingNeeded) + "┃"

	borderBarThick := strings.Repeat("━", cardWidth+1)
	bannerBorderTop := "┏" + borderBarThick + "┓"

	bannerDisplay := bannerBorderTop + "\n" + bannerTxt

	// Initial message / Previous attempts
	// Shown before the board
	var introMsg string
	if g.State.Score.GetAttempts() > 0 {
		introMsg = fmt.Sprintf("\nAttempt: %d | High score (this text): %d\n", g.State.Score.GetAttempts()+1, g.State.Score.GetHighScore().Score)
	} else {
		introMsg = "\nThis is your first try with this text! Good luck!\n"
	}

	// 2. Render Board
	customBorder := lipgloss.ThickBorder()
	customBorder.Top = "═"
	customBorder.TopLeft = "┃"
	customBorder.TopRight = "┃"

	borderStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Border(customBorder).
		Width(cardWidth + 1) // Match manual header width

	display := introMsg + "\n" + bannerDisplay + "\n" + borderStyle.Render(s.RenderBoard())

	// 3. Status Line
	displayScore := g.State.Score.CurrentScore
	if displayScore < 0 {
		displayScore = 0
	}

	statusLine := "SCORE: " + fmt.Sprint(displayScore) + " | " +
		"HINTS: " + fmt.Sprint(g.State.Score.HintCount) + " | " +
		"ERRORS: " + fmt.Sprint(g.State.Score.ErrorCount)

	// Batch Mode Indicator
	if s.Session.IsBatch {
		statusLine += fmt.Sprintf(" | CARD %d/%d", s.Session.CurrentIndex+1, len(s.Session.Cards))
		statusLine += fmt.Sprintf(" | TOTAL: %d", s.Session.TotalScore)
	}

	if g.State.TimerEnabled {
		timeColor := lipgloss.Color("11")

		totalLimit := float64(g.State.TimeLimit)
		// If batch, we want "1/3 of ORIGINAL total time".
		// Session has TotalTimeLimit (original sum).
		if s.Session.IsBatch && s.Session.TotalTimeLimit > 0 {
			totalLimit = float64(s.Session.TotalTimeLimit)
		}

		// Use Game TimeRemaining (which is synced to session)
		if float64(g.State.TimeRemaining) <= totalLimit/3.0 {
			timeColor = lipgloss.Color("9")
		}

		timeStyle := lipgloss.NewStyle().Foreground(timeColor)
		minutes := g.State.TimeRemaining / 60
		seconds := g.State.TimeRemaining % 60
		timeStr := fmt.Sprintf("%02d:%02d", minutes, seconds)
		statusLine += " | TIME: " + timeStyle.Render(timeStr)
	}

	display += "\n" + scoreStyle.Render(statusLine+"\n")

	// Final Messages (Loss/Win)
	if g.State.Loss {
		finalScore := g.State.Score.CurrentScore
		if finalScore < 0 {
			finalScore = 0
		}
		scoreStr := fmt.Sprintf("Final score: %d", finalScore)

		if g.State.Revealed {
			display += "\n" + redStyle.Render("Card revealed with CTRL-R! "+scoreStr) + "\n"
		} else if g.State.TimerEnabled && g.State.TimeRemaining <= 0 {
			display += "\n" + redStyle.Render("Time's up! "+scoreStr) + "\n"
		} else {
			display += "\n" + redStyle.Render("Game over! "+scoreStr) + "\n"
		}
	} else if g.State.Win {
		// Use IsLastGame for the final batch message
		if s.Session.IsLastGame() {
			if s.Session.IsBatch {
				display += "\n" + greenStyle.Render(fmt.Sprintf("Batch Complete! Total Score: %d", s.Session.TotalScore)) + "\n"
			} else {
				display += "\n" + greenStyle.Render(fmt.Sprintf("Congratulations! Final score: %d", g.State.Score.CurrentScore)) + "\n"
				if g.State.Score.GotHighScore() {
					display += "\nYou got a high score!"
					numPrevious := g.State.Score.GetNumPrevious()
					if numPrevious > 0 {
						if numPrevious <= 5 {
							display += " Previous scores:"
						} else {
							display += " Top 5 previous scores:"
						}
						topScores := g.State.Score.GetNScoreEntries(5)
						for _, entry := range topScores {
							display += fmt.Sprintf("\n  * %d on %s", entry.Score, entry.Timestamp)
						}
					}
					display += "\n"
				}
			}
		} else {
			// Intermediate card in batch
			display += "\n" + greenStyle.Render(fmt.Sprintf("Congratulations! Card Score: %d", g.State.Score.CurrentScore)) + "\n"
		}
	}

	return display
}

func capitalize(word string) string {
	if len(word) == 0 {
		return word
	}
	return strings.ToUpper(string(word[0])) + word[1:]
}

func titleCaseToTitle(input string) string {
	var result strings.Builder
	lastCharType := 0 // 0: none, 1: letter, 2: digit

	for i, r := range input {
		currentCharType := 0
		if unicode.IsUpper(r) {
			currentCharType = 1
		} else if unicode.IsLower(r) {
			currentCharType = 1
		} else if unicode.IsDigit(r) {
			currentCharType = 2
		}

		if i > 0 && ((lastCharType == 1 && currentCharType == 2) || (lastCharType == 2 && currentCharType == 1) || (unicode.IsUpper(r) && unicode.IsLower(rune(input[i-1])))) {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
		lastCharType = currentCharType
	}

	// Capitalize each word
	words := strings.Fields(result.String())
	for i, word := range words {
		words[i] = capitalize(word)
	}

	return strings.Join(words, " ")
}

func longestLineLen(s string) int {
	maxLength := 0
	for _, line := range strings.Split(s, "\n") {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}
	return maxLength
}

type timerFlag int

func (t *timerFlag) String() string {
	if *t == -1 {
		return "auto"
	}
	return fmt.Sprint(int(*t))
}

func (t *timerFlag) Set(s string) error {
	if s == "true" {
		*t = -1 // Auto
		return nil
	}
	if s == "false" {
		*t = 0 // Disabled
		return nil
	}

	// Try parsing as simple integer first
	if val, err := strconv.Atoi(s); err == nil {
		*t = timerFlag(val)
		return nil
	}

	// Try parsing MM:SS
	parts := strings.Split(s, ":")
	if len(parts) == 2 {
		min, err1 := strconv.Atoi(parts[0])
		sec, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			*t = timerFlag(min*60 + sec)
			return nil
		}
	}

	return fmt.Errorf("invalid timer format: %s (use 'MM:SS' or seconds)", s)
}

func (t *timerFlag) IsBoolFlag() bool { return true }

type strictIntFlag int

func (i *strictIntFlag) String() string {
	return fmt.Sprint(int(*i))
}

func (i *strictIntFlag) Set(s string) error {
	if s == "true" {
		return fmt.Errorf("value required (format: -flag=value)")
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = strictIntFlag(v)
	return nil
}

func (i *strictIntFlag) IsBoolFlag() bool { return true }

func main() {
	// defaults
	var tFlag timerFlag = -1 // Default to auto
	var noTimer bool
	var firstLetter bool
	var nRandom strictIntFlag
	var nWords strictIntFlag
	var randomCards bool

	// Timer flags
	flag.Var(&tFlag, "timer", "Set countdown timer (e.g. 30 or 1:30). Default is auto based on length.")
	flag.Var(&tFlag, "t", "Set countdown timer (shorthand)")

	flag.BoolVar(&noTimer, "notimer", false, "Disable the timer")
	flag.BoolVar(&noTimer, "nt", false, "Disable the timer (shorthand)")

	// Game mode flags
	flag.BoolVar(&firstLetter, "first-letter", false, "Reveal the first letter of each word")
	flag.BoolVar(&firstLetter, "fl", false, "Reveal the first letter of each word (shorthand)")

	flag.Var(&nRandom, "n-random", "Reveal N random letters")
	flag.Var(&nRandom, "nr", "Reveal N random letters (shorthand)")

	flag.Var(&nWords, "n-words", "Reveal N random words")
	flag.Var(&nWords, "nfw", "Reveal N random words (shorthand)")

	flag.BoolVar(&randomCards, "random-cards", false, "Randomize presentation order of cards")
	flag.BoolVar(&randomCards, "rc", false, "Randomize presentation order of cards (shorthand)")
	flag.BoolVar(&randomCards, "random", false, "Randomize presentation order of cards (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <path-to-file> [more files...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "    -t, --timer[=value]    Set countdown timer (e.g. 30 or 1:30). Default is auto based on length.\n")
		fmt.Fprintf(os.Stderr, "   -nt, --notimer          Disable the timer\n")
		fmt.Fprintf(os.Stderr, "   -fl, --first-letter     Reveal the first letter of each word\n")
		fmt.Fprintf(os.Stderr, "   -nr, --n-random=N       Reveal N random letters\n")
		fmt.Fprintf(os.Stderr, "  -nfw, --n-words=N        Reveal N random words\n")
		fmt.Fprintf(os.Stderr, "   -rc, --random-cards     Randomize order of cards (Batch Mode only)\n")
		fmt.Fprintf(os.Stderr, "    -h, --help             Show this help message\n")
	}

	flag.Parse()

	// Get non-flag arguments
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		return
	}

	// Determine effective timer limit
	timerLimit := int(tFlag)
	if noTimer {
		timerLimit = 0
	}

	opts := state.GameOptions{
		TimerLimit:  timerLimit,
		FirstLetter: firstLetter,
		NRandom:     int(nRandom),
		NWords:      int(nWords),
	}

	// Create the initial model
	model, err := initialModel(args, opts, randomCards)
	if err != nil {
		fmt.Printf("Error initializing model: %v\n", err)
		os.Exit(1)
	}

	// Main Loop: Run one program per card
	session := model.Session
	for {
		// Create a fresh model wrapper for the current session state
		currentModel := &LocalState{
			Session: session,
		}

		p := tea.NewProgram(currentModel)
		_, err := p.Run()
		if err != nil {
			fmt.Printf("Error starting the program: %v\n", err)
			break
		}

		// Check for loss
		if session.IsSessionLoss() {
			// If revealed (gave up), continue to next card. Otherwise (timer), end session.
			if !session.CurrentGame.State.Revealed {
				break
			}
		}

		// Advance to next card
		session.CurrentIndex++
		if session.IsFinished() {
			break
		}

		// Prepare next game
		if err := session.NextGame(); err != nil {
			fmt.Printf("Error preparing next game: %v\n", err)
			break
		}
	}
}
