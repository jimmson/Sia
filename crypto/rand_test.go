package crypto

import (
	"bytes"
	"compress/gzip"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"
)

// panics returns true if the function fn panicked.
func panics(fn func()) (panicked bool) {
	defer func() {
		panicked = (recover() != nil)
	}()
	fn()
	return
}

// TestRandIntnPanics tests that RandIntn panics if n <= 0.
func TestRandIntnPanics(t *testing.T) {
	// Test n = 0.
	if !panics(func() { RandIntn(0) }) {
		t.Error("expected panic for n <= 0")
	}

	// Test n < 0.
	if !panics(func() { RandIntn(-1) }) {
		t.Error("expected panic for n <= 0")
	}
}

// TestRandIntn tests the RandIntn function.
func TestRandIntn(t *testing.T) {
	const iters = 10000
	var counts [10]int
	for i := 0; i < iters; i++ {
		counts[RandIntn(len(counts))]++
	}
	exp := iters / len(counts)
	lower, upper := exp-(exp/10), exp+(exp/10)
	for i, n := range counts {
		if !(lower < n && n < upper) {
			t.Errorf("Expected range of %v-%v for index %v, got %v", lower, upper, i, n)
		}
	}
}

// TestRead tests that Read produces output with sufficiently high entropy.
func TestRead(t *testing.T) {
	const size = 10e3

	var b bytes.Buffer
	zip, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if _, err := zip.Write(RandBytes(size)); err != nil {
		t.Fatal(err)
	}
	if err := zip.Close(); err != nil {
		t.Fatal(err)
	}
	if b.Len() < size {
		t.Error("supposedly high entropy bytes have been compressed!")
	}
}

// TestRandConcurrent checks that there are no race conditions when using the
// rngs concurrently.
func TestRandConcurrent(t *testing.T) {
	// Spin up a thread calling each of the functions offered by the rand
	// package.
	closeChan := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Read some random data into a byte slice.
			buf := make([]byte, 32)
			Read(buf)
		}
	}()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Read some random data into a large byte slice.
			buf := make([]byte, 16e3)
			Read(buf)
		}
	}()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Call io.Copy on the global reader.
			buf := new(bytes.Buffer)
			io.CopyN(buf, Reader, 16e3)
		}
	}()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Call RandIntn
			_ = RandIntn(250)
		}
	}()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Call RandBigIntn
			b := big.NewInt(1e16)
			b = b.Mul(b, b)
			b = b.Mul(b, b)
			_ = RandBigIntn(b)
		}
	}()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-closeChan:
				wg.Done()
				return
			default:
			}

			// Call Perm
			_ = Perm(150)
		}
	}()

	// Wait for a second.
	time.Sleep(time.Second)

	// Close the channel and wait for everything to clean up.
	close(closeChan)
	wg.Wait()
}

// TestPerm tests the Perm function.
func TestPerm(t *testing.T) {
	chars := "abcde" // string to be permuted
	createPerm := func() string {
		s := make([]byte, len(chars))
		for i, j := range Perm(len(chars)) {
			s[i] = chars[j]
		}
		return string(s)
	}

	// create (factorial(len(chars)) * 100) permutations
	permCount := make(map[string]int)
	for i := 0; i < 12000; i++ {
		permCount[createPerm()]++
	}

	// we should have seen each permutation approx. 100 times
	for p, n := range permCount {
		if n < 50 || n > 150 {
			t.Errorf("saw permutation %v times: %v", n, p)
		}
	}
}
