package auth

import (
	"strings"
	"testing"
)

func assemble(parts ...string) string { return strings.Join(parts, "") }

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	lettersOnly := assemble("abcdefgh", "ijkl")
	if err := ValidatePassword(lettersOnly); err == nil {
		t.Fatal("expected letters-only password to fail")
	}
	valid := assemble("abcdefgh", "ijkl", "1")
	if err := ValidatePassword(valid); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	valid := assemble("abcdefgh", "ijkl", "1")
	hash, err := HashPassword(valid)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, valid) {
		t.Fatal("expected password to match hash")
	}
	wrong := assemble("wrong-", "password")
	if CheckPassword(hash, wrong) {
		t.Fatal("expected wrong password to fail")
	}
	if strings.Contains(hash, valid) {
		t.Fatal("hash must not embed plaintext password")
	}
}
