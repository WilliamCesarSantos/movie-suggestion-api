package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordService struct {
	pepper string
}

func NewPasswordService(pepper string) *PasswordService {
	return &PasswordService{pepper: pepper}
}

func (s *PasswordService) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	peppered := s.pepper + password
	hash := argon2.IDKey([]byte(peppered), salt, 3, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

func (s *PasswordService) Verify(password, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	peppered := s.pepper + password
	computed := argon2.IDKey([]byte(peppered), salt, 3, 64*1024, 4, uint32(len(expectedHash)))
	if len(computed) != len(expectedHash) {
		return false, nil
	}
	diff := 0
	for i := range computed {
		diff |= int(computed[i] ^ expectedHash[i])
	}
	return diff == 0, nil
}
