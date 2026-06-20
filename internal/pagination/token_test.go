package pagination

import (
	"errors"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	token := Token(25)
	if token == "" {
		t.Fatal("expected token for positive offset")
	}
	offset, err := Offset(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if offset != 25 {
		t.Fatalf("expected offset 25, got %d", offset)
	}
}

func TestOffsetRejectsInvalidToken(t *testing.T) {
	_, err := Offset("not-base64")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenForZeroOffsetIsEmpty(t *testing.T) {
	if token := Token(0); token != "" {
		t.Fatalf("expected empty token for zero offset, got %q", token)
	}
}
