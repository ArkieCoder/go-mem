package game

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CardData struct {
	Content string
	Source  string
}

// LoadCards loads cards from a list of paths (files or directories).
func LoadCards(paths []string) ([]CardData, error) {
	var cards []CardData

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to access path %s: %w", path, err)
		}

		if info.IsDir() {
			// Read directory
			files, err := os.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read dir %s: %w", path, err)
			}
			for _, entry := range files {
				if !entry.IsDir() {
					c, err := loadFile(filepath.Join(path, entry.Name()))
					if err != nil {
						// Optionally warn instead of fail? strict for now.
						return nil, err
					}
					cards = append(cards, c...)
				}
			}
		} else {
			// Read file
			c, err := loadFile(path)
			if err != nil {
				return nil, err
			}
			cards = append(cards, c...)
		}
	}

	return cards, nil
}

func loadFile(path string) ([]CardData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var contentBuilder strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		contentBuilder.WriteString(scanner.Text() + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file %s: %w", path, err)
	}

	content := contentBuilder.String()

	// Split by separator: line starting with 3+ dashes
	// Regex: (?m)^-{3,}\s*$
	// Note: We need to handle potential split at EOF?

	separatorRe := regexp.MustCompile(`(?m)^-{3,}[ \t]*$`)
	parts := separatorRe.Split(content, -1)

	var cards []CardData
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if len(trimmed) > 0 {
			cards = append(cards, CardData{
				Content: trimmed,
				Source:  path,
			})
		}
	}

	return cards, nil
}
