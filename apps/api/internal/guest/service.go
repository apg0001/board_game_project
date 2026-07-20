package guest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

type Clock func() time.Time

type Service struct {
	store Store
	clock Clock
}

func NewService(store Store, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: store, clock: clock}
}

func (s *Service) Create() (User, error) {
	now := s.clock().UTC()
	nickname, err := s.uniqueNickname()
	if err != nil {
		return User{}, err
	}

	token, err := secureToken(24)
	if err != nil {
		return User{}, err
	}

	id, err := secureToken(12)
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:           "guest_" + id,
		Nickname:     nickname,
		SessionToken: token,
		CreatedAt:    now,
		LastSeenAt:   now,
	}

	if err := s.store.Save(user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) Me(token string) (User, error) {
	user, err := s.store.FindByToken(token)
	if err != nil {
		return User{}, err
	}

	user.LastSeenAt = s.clock().UTC()
	if err := s.store.Save(user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) uniqueNickname() (string, error) {
	for range 20 {
		number, err := randomInt(1000, 9999)
		if err != nil {
			return "", err
		}
		nickname := fmt.Sprintf("Guest_%d", number)
		if !s.store.NicknameExists(nickname) {
			return nickname, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique guest nickname")
}

func randomInt(minimum int64, maximum int64) (int64, error) {
	diff := maximum - minimum + 1
	value, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return 0, err
	}
	return minimum + value.Int64(), nil
}

func secureToken(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
