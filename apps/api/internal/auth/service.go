package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrDuplicateUsername = errors.New("username already exists")
	ErrInvalidCredential = errors.New("invalid username or password")
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type Account struct {
	User
	Salt         string
	PasswordHash string
}

type Clock func() time.Time

type Store interface {
	FindAccountByUsername(username string) (Account, bool)
	SaveAccount(account Account) error
	SaveSession(token string, user User) error
	FindSession(token string) (User, bool)
}

type Service struct {
	clock Clock
	store Store
}

func NewService(clock Clock) *Service {
	return NewServiceWithStore(NewMemoryStore(), clock)
}

func NewServiceWithStore(store Store, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{clock: clock, store: store}
}

func (s *Service) Register(username string, password string, nickname string) (User, string, error) {
	username = normalize(username)
	nickname = displayNickname(nickname, username)
	if nickname == "" {
		nickname = username
	}
	if username == "" || password == "" {
		return User{}, "", ErrInvalidCredential
	}

	if _, ok := s.store.FindAccountByUsername(username); ok {
		return User{}, "", ErrDuplicateUsername
	}

	salt := randomHex(16)
	user := User{
		ID:        "user_" + randomHex(12),
		Username:  username,
		Nickname:  nickname,
		Role:      "USER",
		CreatedAt: s.clock().UTC(),
	}
	account := Account{
		User:         user,
		Salt:         salt,
		PasswordHash: passwordHash(salt, password),
	}
	if err := s.store.SaveAccount(account); err != nil {
		return User{}, "", err
	}
	token := randomHex(24)
	if err := s.store.SaveSession(token, user); err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func (s *Service) EnsureAdmin(username string, password string, nickname string) (User, error) {
	username = normalize(username)
	nickname = displayNickname(nickname, username)
	if username == "" || password == "" {
		return User{}, ErrInvalidCredential
	}

	now := s.clock().UTC()
	account, ok := s.store.FindAccountByUsername(username)
	if !ok {
		account = Account{
			User: User{
				ID:        "user_" + randomHex(12),
				Username:  username,
				Nickname:  nickname,
				Role:      "ADMIN",
				CreatedAt: now,
			},
		}
	}

	account.Nickname = nickname
	account.Role = "ADMIN"
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.Salt = randomHex(16)
	account.PasswordHash = passwordHash(account.Salt, password)
	if err := s.store.SaveAccount(account); err != nil {
		return User{}, err
	}
	return account.User, nil
}

func (s *Service) Login(username string, password string) (User, string, error) {
	username = normalize(username)

	account, ok := s.store.FindAccountByUsername(username)
	if !ok || account.PasswordHash != passwordHash(account.Salt, password) {
		return User{}, "", ErrInvalidCredential
	}

	token := randomHex(24)
	user := normalizeUser(account.User)
	if err := s.store.SaveSession(token, user); err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func (s *Service) Me(token string) (User, bool) {
	user, ok := s.store.FindSession(token)
	if !ok {
		return User{}, false
	}
	return normalizeUser(user), true
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeUser(user User) User {
	user.Nickname = displayNickname(user.Nickname, user.Username)
	if user.Role == "" {
		user.Role = "USER"
	}
	return user
}

func displayNickname(nickname string, fallback string) string {
	nickname = strings.TrimSpace(nickname)
	if nickname != "" {
		return nickname
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return "User"
}

func passwordHash(salt string, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func randomHex(byteLength int) string {
	buffer := make([]byte, byteLength)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}
