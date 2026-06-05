package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	if err := ValidatePassword("abcdefghijkl"); err == nil {
		t.Fatal("expected letters-only password to fail")
	}
	if err := ValidatePassword("abcdefghijkl1"); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("abcdefghijkl1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "abcdefghijkl1") {
		t.Fatal("expected password to match hash")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail")
	}
}
