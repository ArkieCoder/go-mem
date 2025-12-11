package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCards_SingleFile(t *testing.T) {
	content := "Card 1\nLine 2"
	path := createTempFile(t, content)
	defer os.Remove(path)

	cards, err := LoadCards([]string{path})
	if err != nil {
		t.Fatalf("LoadCards failed: %v", err)
	}

	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}
	if cards[0].Content != content {
		t.Errorf("Content mismatch. Got %q", cards[0].Content)
	}
}

func TestLoadCards_MultipleCardsInFile(t *testing.T) {
	content := `Card 1
---
Card 2
----------------
Card 3`
	path := createTempFile(t, content)
	defer os.Remove(path)

	cards, err := LoadCards([]string{path})
	if err != nil {
		t.Fatalf("LoadCards failed: %v", err)
	}

	if len(cards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(cards))
	}
	if cards[0].Content != "Card 1" {
		t.Errorf("Card 1 mismatch: %q", cards[0].Content)
	}
	if cards[1].Content != "Card 2" {
		t.Errorf("Card 2 mismatch: %q", cards[1].Content)
	}
	if cards[2].Content != "Card 3" {
		t.Errorf("Card 3 mismatch: %q", cards[2].Content)
	}
}

func TestLoadCards_Directory(t *testing.T) {
	dir, err := os.MkdirTemp("", "card_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file1 := filepath.Join(dir, "f1.txt")
	os.WriteFile(file1, []byte("C1"), 0644)

	file2 := filepath.Join(dir, "f2.txt")
	os.WriteFile(file2, []byte("C2\n---\nC3"), 0644)

	cards, err := LoadCards([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	// Order is not guaranteed by ReadDir usually, but let's check count
	if len(cards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(cards))
	}
}

func createTempFile(t *testing.T, content string) string {
	f, err := os.CreateTemp("", "card_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}
