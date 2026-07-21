package jokerdraw

import "testing"

func TestRemovePairsKeepsUnmatchedAndJoker(t *testing.T) {
	hand := []Card{{ID: "A-1", Rank: "A"}, {ID: "A-2", Rank: "A"}, {ID: "K-1", Rank: "K"}, {ID: "joker", Rank: "joker", Joker: true}}
	result := removePairs(hand)
	if len(result) != 2 {
		t.Fatalf("expected unmatched king and joker, got %d", len(result))
	}
}

func TestShuffledDeckHasJoker(t *testing.T) {
	deck := shuffledDeck("seed")
	if len(deck) != 53 {
		t.Fatalf("expected 53 cards, got %d", len(deck))
	}
	found := false
	for _, card := range deck {
		found = found || card.Joker
	}
	if !found {
		t.Fatal("expected joker")
	}
}
