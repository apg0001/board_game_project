package auth

import "testing"

func TestRegisterLoginAndMe(t *testing.T) {
	service := NewService(nil)

	created, token, err := service.Register("Alice", "secret", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if created.Username != "alice" || token == "" {
		t.Fatalf("unexpected register result: %+v %s", created, token)
	}

	loggedIn, loginToken, err := service.Login("alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.ID != created.ID || loginToken == "" {
		t.Fatal("expected same user and login token")
	}

	me, ok := service.Me(loginToken)
	if !ok || me.ID != created.ID {
		t.Fatal("expected session lookup")
	}
}

func TestDuplicateRegister(t *testing.T) {
	service := NewService(nil)
	if _, _, err := service.Register("alice", "secret", "Alice"); err != nil {
		t.Fatal(err)
	}
	_, _, err := service.Register("alice", "secret", "Alice")
	if err != ErrDuplicateUsername {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}
