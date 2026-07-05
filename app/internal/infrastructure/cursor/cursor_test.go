package cursor

import (
	"errors"
	"strings"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	secret := "test-secret"
	encoded := Encode(secret, Cursor{Offset: 10, Total: 45})
	if encoded == "" {
		t.Fatal("expected non-empty encoded cursor")
	}

	decoded, err := Decode(secret, encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.Offset != 10 || decoded.Total != 45 {
		t.Fatalf("unexpected decoded cursor = %+v", decoded)
	}
}

func TestDecodeRejectsTamperedCursor(t *testing.T) {
	secret := "test-secret"
	encoded := Encode(secret, Cursor{Offset: 10, Total: 45})
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		t.Fatalf("expected two cursor parts, got %d", len(parts))
	}

	tampered := parts[0] + ".AAAA"
	_, err := Decode(secret, tampered)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestDecodeRejectsMalformedCursor(t *testing.T) {
	_, err := Decode("test-secret", "not-a-cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}
