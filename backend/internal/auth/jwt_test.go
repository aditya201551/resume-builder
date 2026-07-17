package auth

import (
	"testing"
	"time"
)

func TestJWTIssueAndParse(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", time.Hour)

	token, err := issuer.Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("UserID = %q, want %q", claims.UserID, "user-123")
	}
}

func TestJWTExpired(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", -time.Hour)

	token, err := issuer.Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := issuer.Parse(token); err == nil {
		t.Fatal("Parse() expected error for expired token, got nil")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	token, err := NewJWTIssuer("secret-a", time.Hour).Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := NewJWTIssuer("secret-b", time.Hour).Parse(token); err == nil {
		t.Fatal("Parse() expected error for wrong secret, got nil")
	}
}

func TestStateSignerRoundTrip(t *testing.T) {
	signer := NewStateSigner("test-secret", time.Minute)

	state, err := signer.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := signer.Verify(state); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestStateSignerExpired(t *testing.T) {
	signer := NewStateSigner("test-secret", -time.Minute)

	state, err := signer.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := signer.Verify(state); err == nil {
		t.Fatal("Verify() expected error for expired state, got nil")
	}
}

func TestStateSignerTampered(t *testing.T) {
	signer := NewStateSigner("test-secret", time.Minute)

	state, err := signer.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := signer.Verify(state + "x"); err == nil {
		t.Fatal("Verify() expected error for tampered state, got nil")
	}
}
