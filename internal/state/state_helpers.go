package state

import (
	"regexp"
	"slices"
	"strings"
)

func (s *State) SetBracketedPositions() {
	bracketContentsRe := regexp.MustCompile(`(?s)\[(.*?)\]`)
	bracketRe := regexp.MustCompile(`[^\[\]]`)
	secretStr := string(s.Secret)
	matches := bracketContentsRe.FindAllStringSubmatchIndex(secretStr, -1)

	var positions []int
	for _, match := range matches {
		startMatch := match[2]
		endMatch := match[3]
		for i := startMatch; i < endMatch; i++ {
			stringUpToMatch := secretStr[:startMatch]
			previousBrackets := len(bracketRe.ReplaceAllString(stringUpToMatch, ""))
			positions = append(positions, i-previousBrackets)
		}
	}
	s.BracketedPositions = positions
	s.Secret = []rune(bracketContentsRe.ReplaceAllString(secretStr, "$1"))
}

func (s *State) InitMask() {
	mask := make([]rune, len(s.Secret))

	for i, ch := range s.Secret {
		if s.ShouldIgnore(string(ch)) || slices.Contains(s.BracketedPositions, i) {
			mask[i] = ch
		} else {
			mask[i] = '_'
		}
	}
	s.Mask = mask
}

func isPunctuation(r rune) bool {
	return strings.ContainsRune(",.!?;:\n", r)
}

func IsExitRequested(ch string) bool {
	return ch == "ctrl+c"
}

func IsRevealRequested(ch string) bool {
	return ch == "ctrl+r"
}

func IsTabRequested(ch string) bool {
	return ch == "tab"
}

func (s State) ShouldIgnore(ch string) bool {
	if len(ch) == 0 {
		return false
	}

	isSpace := ch == " "
	isNonQuestionMarkPunc := (isPunctuation(rune(ch[0])) && ch != "?")

	return isSpace || isNonQuestionMarkPunc
}

func (s State) IsAtEnd() bool {
	return s.Pos == len(s.Secret)
}

func (s State) IsCorrectLetter(ch string) bool {
	if s.Pos >= len(s.Secret) {
		return false
	}
	return strings.ToLower(ch) == strings.ToLower(string(s.Secret[s.Pos]))
}

func (s *State) IsIncorrectLetter(ch string) bool {
	if s.Pos >= len(s.Secret) {
		return true
	}
	return strings.ToLower(ch) != strings.ToLower(string(s.Secret[s.Pos]))
}

func (s State) GotCompletedWord() bool {
	return !s.IsAtEnd() &&
		(s.Secret[s.Pos] == ' ' || isPunctuation(s.Secret[s.Pos]))
}

func (s State) GotCorrectMessage() bool {
	return string(s.Secret) == s.Textarea.Value()
}

func (s State) IsGameOver() bool {
	return (s.Pos >= len(s.Secret)) || s.Score.CurrentScore < 0
}

func (s State) LostGame() bool {
	return s.Score.CurrentScore < 0 || (s.IsGameOver() && s.WrongLetter)
}

func (s State) WonGame() bool {
	return !s.LostGame()
}
