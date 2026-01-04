package random

import (
	"crypto/rand"
	"math/big"
)

// Random provides random number generation that can be mocked for testing
type Random interface {
	// Intn returns a random int in [0, n)
	Intn(n int) int

	// String generates a random string of the given length from the given alphabet
	String(length int, alphabet string) string
}

// CryptoRandom implements Random using crypto/rand
type CryptoRandom struct{}

// New creates a new CryptoRandom
func New() *CryptoRandom {
	return &CryptoRandom{}
}

// Intn returns a cryptographically random int in [0, n)
func (r *CryptoRandom) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fall back to 0 on error (should never happen with crypto/rand)
		return 0
	}
	return int(result.Int64())
}

// String generates a random string of the given length from the given alphabet
func (r *CryptoRandom) String(length int, alphabet string) string {
	if length <= 0 || len(alphabet) == 0 {
		return ""
	}
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(result)
}
