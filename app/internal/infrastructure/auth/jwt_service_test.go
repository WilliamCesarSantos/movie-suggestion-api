package auth

import "testing"

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService("secret", 1)

	token, _, err := svc.Generate("user-1", "user@example.com", []string{"users:read", "*"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if claims.Subject != "user-1" {
		t.Fatalf("expected subject user-1, got %s", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", claims.Email)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "users:read" || claims.Roles[1] != "*" {
		t.Fatalf("unexpected roles: %#v", claims.Roles)
	}
}

func TestJWTService_ValidateRejectsInvalidToken(t *testing.T) {
	svc := NewJWTService("secret", 1)
	if _, err := svc.Validate("invalid"); err == nil {
		t.Fatal("expected validation error")
	}
}
