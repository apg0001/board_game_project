package sutda

import "testing"

func TestEvaluateRanks(t *testing.T) {
	rank, name := evaluate([]Card{{Month: 3, Gwang: true}, {Month: 8, Gwang: true}})
	if name != "38광땡" || rank < 1000 {
		t.Fatalf("expected 38 gwang, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 1}, {Month: 2}})
	if name != "알리" || rank != 700 {
		t.Fatalf("expected ali, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 4}, {Month: 5}})
	if name != "갑오" || rank != 609 {
		t.Fatalf("expected gap-o, got %s %d", name, rank)
	}
}

func TestCompareHands(t *testing.T) {
	left := []Card{{Month: 10}, {Month: 10}}
	right := []Card{{Month: 1}, {Month: 2}}
	if compare(left, right) <= 0 {
		t.Fatal("expected jang-ttaeng to beat ali")
	}
}
