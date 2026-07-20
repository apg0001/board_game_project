package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
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
	CreatedAt time.Time `json:"createdAt"`
}

type account struct {
	User
	Salt         string
	PasswordHash string
}

type Clock func() time.Time

type Service struct {
	mu      sync.RWMutex
	clock   Clock
	byName  map[string]account
	byToken map[string]User
}

func NewService(clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		clock:   clock,
		byName:  make(map[string]account),
		byToken: make(map[string]User),
	}
}

func (s *Service) Register(username string, password string, nickname string) (User, string, error) {
	username = normalize(username)
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = username
	}
	if username == "" || password == "" {
		return User{}, "", ErrInvalidCredential
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byName[username]; ok {
		return User{}, "", ErrDuplicateUsername
	}

	salt := randomHex(16)
	user := User{
		ID:        "user_" + randomHex(12),
		Username:  username,
		Nickname:  nickname,
		CreatedAt: s.clock().UTC(),
	}
	s.byName[username] = account{
		User:         user,
		Salt:         salt,
		PasswordHash: passwordHash(salt, password),
	}
	token := randomHex(24)
	s.byToken[token] = user
	return user, token, nil
}

func (s *Service) Login(username string, password string) (User, string, error) {
	username = normalize(username)
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.byName[username]
	if !ok || account.PasswordHash != passwordHash(account.Salt, password) {
		return User{}, "", ErrInvalidCredential
	}

	token := randomHex(24)
	s.byToken[token] = account.User
	return account.User, token, nil
}

func (s *Service) Me(token string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.byToken[token]
	return user, ok
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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
