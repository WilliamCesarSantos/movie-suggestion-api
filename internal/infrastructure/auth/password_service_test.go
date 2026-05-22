package auth

import "testing"

func TestPasswordService_HashAndVerify(t *testing.T) {
	svc := NewPasswordService("pepper")

	hash, err := svc.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	ok, err := svc.Verify("password123", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("expected password verification to succeed")
	}

	ok, err = svc.Verify("wrong", hash)
	if err != nil {
		t.Fatalf("Verify() with wrong password error = %v", err)
	}
	if ok {
		t.Fatal("expected password verification to fail")
	}
}

func TestPasswordService_VerifyRejectsInvalidFormat(t *testing.T) {
	svc := NewPasswordService("pepper")
	if _, err := svc.Verify("password123", "invalid"); err == nil {
		t.Fatal("expected invalid format error")
	}
}
