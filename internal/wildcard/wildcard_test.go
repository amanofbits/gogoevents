package wildcard_test

import (
	"testing"

	"github.com/amanofbits/gogoevents/internal/wildcard"
)

func TestNormalize(t *testing.T) {
	s := []string{
		"",
		"test",
		"t",
		"**",
		"*big*test*",
		"**big******test***2*2",
	}
	ans := []string{
		"",
		"test",
		"t",
		"*",
		"*big*test*",
		"*big*test*2*2",
	}

	for i := range s {
		a := wildcard.Normalize(s[i])
		if ans[i] != a {
			t.Fatalf("expected %s, got %s", ans[i], a)
		}
	}
}
