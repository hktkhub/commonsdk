package ratelimit

import "testing"

func TestJoinDoesNotCollide(t *testing.T) {
	if Join("a", "bc") == Join("ab", "c") {
		t.Fatal("Join produced colliding keys")
	}
	if got, want := Join("ip", "1.2.3.4", "user", "alice"), "2:ip7:1.2.3.44:user5:alice"; got != want {
		t.Fatalf("Join = %q, want %q", got, want)
	}
}
