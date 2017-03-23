package crypto

import (
	"crypto/rand"
	"io"
	"math"
	"math/big"
	"runtime"
	"unsafe"
)

// randReader reads entropy from the package's global entropy pool.
type randReader struct {}

// entropyChan holds a buffer of 32kb of entropy, so that entropy can be served
// quickly and restored in the background. Entropy can be refilled in parallel.
var entropyChan = make(chan Hash, 1e3)

// Reader is a global, shared instance of a cryptographically strong pseudo-
// random generator. Reader is safe for concurrent use by multiple goroutines.
var Reader = &randReader{}

// init creates workers that continuously fill the entropy pool.
func init() {
	for i := 0; i < runtime.NumCPU(); i++ {
		go threadedFillEntropy()
	}
}

// threadedFillEntropy keeps a hasher and uses it to continually fill the
// entropy channel with entropy.
func threadedFillEntropy() {
	// Get a hasher and fill it with 64 bytes of entropy. Technically only 16
	// should be needed, but the underlying rng may not be secure.
	h := NewHash()
	n, err := io.CopyN(h, rand.Reader, 64)
	if err != nil || n != 64 {
		panic("crypto: no entropy available")
	}
	seed := h.Sum(nil)

	for {
		for i := uint64(0); i < math.MaxUint64; i++ {
			// Update the seed.
			*(*uint64)(unsafe.Pointer(&seed[0])) = i

			// Reset the hasher and get new entropy.
			var result Hash
			h.Reset()
			h.Write(seed[:])
			h.Sum(result[:0])

			// Send the entropy down the entropy channel.
			entropyChan <- result
		}

		// Re-seed the hasher. Use the entropy that existed previously,
		// protecting against a compromised rng.
		h.Reset()
		h.Write(seed[:])
		io.CopyN(h, rand.Reader, 64)
		seed = h.Sum(nil)
	}
}

// Read fills b with random data. It always returns len(b), nil.
func (r *randReader) Read(b []byte) (int, error) {
	n := 0
	for n < len(b) {
		entropy := <-entropyChan
		n += copy(b, entropy[:])
	}
	return n, nil
}

// Read is a helper function that calls Reader.Read on b. It always fills b
// completely.
func Read(b []byte) { Reader.Read(b) }

// Bytes is a helper function that returns n bytes of random data.
func RandBytes(n int) []byte {
	b := make([]byte, n)
	Read(b)
	return b
}

// RandIntn returns a uniform random value in [0,n). It panics if n <= 0.
func RandIntn(n int) int {
	if n <= 0 {
		panic("crypto: argument to Intn is <= 0")
	}
	// To eliminate modulo bias, keep selecting at random until we fall within
	// a range that is evenly divisible by n.
	// NOTE: since n is at most math.MaxUint64/2, max is minimized when:
	//    n = math.MaxUint64/4 + 1 -> max = math.MaxUint64 - math.MaxUint64/4
	// This gives an expected 1.333 tries before choosing a value < max.
	max := math.MaxUint64 - math.MaxUint64%uint64(n)
	b := RandBytes(8)
	r := *(*uint64)(unsafe.Pointer(&b[0]))
	for r >= max {
		Read(b)
		r = *(*uint64)(unsafe.Pointer(&b[0]))
	}
	return int(r % uint64(n))
}

// RandBigIntn returns a uniform random value in [0,n). It panics if n <= 0.
func RandBigIntn(n *big.Int) *big.Int {
	i, _ := rand.Int(Reader, n)
	return i
}

// Perm returns a random permutation of the integers [0,n).
func Perm(n int) []int {
	m := make([]int, n)
	for i := 1; i < n; i++ {
		j := RandIntn(i + 1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}
