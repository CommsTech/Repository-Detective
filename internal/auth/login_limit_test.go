package auth

import "testing"

func TestLoginLimiterAllowsBurstThenBlocks(t *testing.T) {
	lim := NewLoginLimiter(1, 3, 64)
	key := "10.0.0.1"
	for i := 0; i < 3; i++ {
		if !lim.Allow(key) {
			t.Fatalf("burst slot %d should allow", i)
		}
	}
	if lim.Allow(key) {
		t.Fatal("expected rate limit after burst")
	}
	if !lim.Allow("10.0.0.2") {
		t.Fatal("other client should still be allowed")
	}
}
