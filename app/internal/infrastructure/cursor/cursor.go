package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Cursor struct {
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

func Encode(secret string, c Cursor) string {
	payload, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	sig := sign(secret, payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func Decode(secret string, token string) (*Cursor, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidCursor
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	if !hmac.Equal(sig, sign(secret, payload)) {
		return nil, ErrInvalidCursor
	}

	var cursor Cursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}
	if cursor.Offset < 0 || cursor.Total < 0 {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}

func sign(secret string, payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
