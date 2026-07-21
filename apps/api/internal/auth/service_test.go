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
	if created.Role != "USER" {
		t.Fatalf("expected user role, got %s", created.Role)
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

func TestEnsureAdmin(t *testing.T) {
	service := NewService(nil)

	admin, err := service.EnsureAdmin("ccl7103", "ccl7103!", "관리자")
	if err != nil {
		t.Fatal(err)
	}
	if admin.Username != "ccl7103" || admin.Role != "ADMIN" {
		t.Fatalf("unexpected admin user: %+v", admin)
	}

	loggedIn, token, err := service.Login("ccl7103", "ccl7103!")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || loggedIn.Role != "ADMIN" {
		t.Fatalf("expected admin login, got %+v %s", loggedIn, token)
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

func TestRegisterFallbackNickname(t *testing.T) {
	service := NewService(nil)

	created, _, err := service.Register("NoNick", "secret", " ")
	if err != nil {
		t.Fatal(err)
	}
	if created.Nickname != "nonick" {
		t.Fatalf("expected username fallback nickname, got %s", created.Nickname)
	}
}
