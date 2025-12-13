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
	if cards[0].TotalParts != 1 {
		t.Errorf("Expected TotalParts 1, got %d", cards[0].TotalParts)
	}
	if cards[0].PartIndex != 1 {
		t.Errorf("Expected PartIndex 1, got %d", cards[0].PartIndex)
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

	// Check Content and Indexing
	if cards[0].Content != "Card 1" {
		t.Errorf("Card 1 mismatch: %q", cards[0].Content)
	}
	if cards[0].PartIndex != 1 || cards[0].TotalParts != 3 {
		t.Errorf("Card 1 indexing wrong: #%d of %d", cards[0].PartIndex, cards[0].TotalParts)
	}

	if cards[1].Content != "Card 2" {
		t.Errorf("Card 2 mismatch: %q", cards[1].Content)
	}
	if cards[1].PartIndex != 2 || cards[1].TotalParts != 3 {
		t.Errorf("Card 2 indexing wrong: #%d of %d", cards[1].PartIndex, cards[1].TotalParts)
	}

	if cards[2].Content != "Card 3" {
		t.Errorf("Card 3 mismatch: %q", cards[2].Content)
	}
	if cards[2].PartIndex != 3 || cards[2].TotalParts != 3 {
		t.Errorf("Card 3 indexing wrong: #%d of %d", cards[2].PartIndex, cards[2].TotalParts)
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

	if len(cards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(cards))
	}
}

func TestLoadCards_WithTitles(t *testing.T) {
	content := `NAME: Card One
This is the content.
---
NAME: Card Two
Second card content.
---
Third Card (No Name)`
	path := createTempFile(t, content)
	defer os.Remove(path)

	cards, err := LoadCards([]string{path})
	if err != nil {
		t.Fatalf("LoadCards failed: %v", err)
	}

	if len(cards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(cards))
	}

	// Check Title extraction
	if cards[0].Title != "Card One" {
		t.Errorf("Card 1 Title mismatch. Got %q", cards[0].Title)
	}
	if cards[0].PartIndex != 1 || cards[0].TotalParts != 3 {
		t.Errorf("Card 1 indexing wrong")
	}

	if cards[1].Title != "Card Two" {
		t.Errorf("Card 2 Title mismatch. Got %q", cards[1].Title)
	}

	if cards[2].Title != "" {
		t.Errorf("Card 3 Title expected empty, got %q", cards[2].Title)
	}
	if cards[2].PartIndex != 3 || cards[2].TotalParts != 3 {
		t.Errorf("Card 3 indexing wrong")
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
