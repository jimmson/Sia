package crypto

import (
	"crypto/rand"
	"math"
	"sync"
	"testing"
)

// BenchmarkRandIntn benchmarks the RandIntn function for small ints.
func BenchmarkRandIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = RandIntn(4e3)
	}
}

// BenchmarkRandIntnLarge benchmarks the RandIntn function for large ints.
func BenchmarkRandIntnLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// constant chosen to trigger resampling (see RandIntn)
		_ = RandIntn(math.MaxUint64/4 + 1)
	}
}

// BenchmarkRead benchmarks the speed of Read for small slices.
func BenchmarkRead32(b *testing.B) {
	b.SetBytes(32 * 5e3)
	buf := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		for j := 0; j < 5e3; j++ {
			Read(buf)
		}
	}
}

// BenchmarkRead512K benchmarks the speed of Read for larger slices.
func BenchmarkRead512K(b *testing.B) {
	b.SetBytes(512e3)
	buf := make([]byte, 512e3)
	for i := 0; i < b.N; i++ {
		Read(buf)
	}
}

// BenchmarkRead4Threads benchmarks the speed of Read when it's being using
// across four threads.
func BenchmarkRead4Threads(b *testing.B) {
	b.SetBytes(4 * 32 * 5e3)
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(4)
		for i := 0; i < 4; i++ {
			go func() {
				for i := 0; i < 5e3; i++ {
					buf := make([]byte, 32)
					Read(buf)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkRead4Threads512k benchmarks the speed of Read when it's being using
// across four threads with 512kb read sizes.
func BenchmarkRead4Threads512k(b *testing.B) {
	b.SetBytes(4 * 512e3)
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(4)
		for i := 0; i < 4; i++ {
			go func() {
				buf := make([]byte, 512e3)
				Read(buf)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkRead64Threads benchmarks the speed of Read when it's being using
// across four threads.
func BenchmarkRead64Threads(b *testing.B) {
	b.SetBytes(64 * 32 * 5e3)
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(64)
		for i := 0; i < 64; i++ {
			go func() {
				for i := 0; i < 5e3; i++ {
					buf := make([]byte, 32)
					Read(buf)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkRead64Threads512k benchmarks the speed of Read when it's being using
// across four threads with 512kb read sizes.
func BenchmarkRead64Threads512k(b *testing.B) {
	b.SetBytes(64 * 512e3)
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(64)
		for i := 0; i < 64; i++ {
			go func() {
				buf := make([]byte, 512e3)
				Read(buf)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkReadCrypto benchmarks the speed of (crypto/rand).Read for small
// slices. This establishes a lower limit for BenchmarkRead32.
func BenchmarkReadCrypto32(b *testing.B) {
	b.SetBytes(32)
	buf := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		rand.Read(buf)
	}
}

// BenchmarkReadCrypto512K benchmarks the speed of (crypto/rand).Read for larger
// slices. This establishes a lower limit for BenchmarkRead512K.
func BenchmarkReadCrypto512K(b *testing.B) {
	b.SetBytes(512e3)
	buf := make([]byte, 512e3)
	for i := 0; i < b.N; i++ {
		rand.Read(buf)
	}
}

// BenchmarkPerm benchmarks the speed of Perm for small slices.
func BenchmarkPerm32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Perm(32)
	}
}

// BenchmarkPermLarge benchmarks the speed of Perm for large slices.
func BenchmarkPermLarge4k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Perm(4e3)
	}
}
