package crypto

import (
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		pw   string
		want bool
	}{
		{"correct horse battery", true},
		{"correct horse batterY", false},
		{"", false},
	} {
		ok, err := VerifyPassword(tc.pw, hash)
		if err != nil || ok != tc.want {
			t.Errorf("VerifyPassword(%q) = %v, %v; want %v", tc.pw, ok, err, tc.want)
		}
	}
	if NeedsRehash(hash) {
		t.Error("fresh hash should not need rehash")
	}
}

func TestSigner(t *testing.T) {
	a := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "oauth")
	b := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "share")

	v := a.Sign([]byte("hello"), time.Minute)
	if got, err := a.Verify(v); err != nil || string(got) != "hello" {
		t.Fatalf("Verify = %q, %v", got, err)
	}
	if _, err := b.Verify(v); err == nil {
		t.Error("a value signed for one purpose must not verify for another")
	}
	if _, err := a.Verify(v + "x"); err == nil {
		t.Error("tampered value verified")
	}
	if _, err := a.Verify(a.Sign([]byte("x"), -time.Second)); err == nil {
		t.Error("expired value verified")
	}
}
