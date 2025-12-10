package consensus

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

var natoWords = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey",
	"xray", "yankee", "zulu", "trust", "mesh",
}

// WordPass returns a random word-based password: e.g. "alpha-hotel-zulu"
func WordPass(count int) (string, error) {
	if count <= 0 {
		return "", fmt.Errorf("invalid count %d", count)
	}

	words := make([]string, count)
	n := big.NewInt(int64(len(natoWords)))

	for i := 0; i < count; i++ {
		r, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("rand failed: %w", err)
		}
		words[i] = natoWords[r.Int64()]
	}

	return strings.Join(words, "-"), nil
}
