package tutorial

import "testing"

func TestFindGuide(t *testing.T) {
	service := NewService()
	guide, ok := service.Find("davinci")
	if !ok {
		t.Fatal("expected davinci guide")
	}
	if len(guide.Steps) == 0 {
		t.Fatal("expected guide steps")
	}
}
