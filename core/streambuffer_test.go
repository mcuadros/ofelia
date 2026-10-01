package core

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/armon/circbuf"
)

func TestStreamBufferEmpty(t *testing.T) {
	b := newStreamBuffer(1024)
	if got := b.String(); got != "" {
		t.Fatalf("String() = %q, want empty", got)
	}
	if got := b.Bytes(); len(got) != 0 {
		t.Fatalf("Bytes() length = %d, want 0", len(got))
	}
	if got := b.TotalWritten(); got != 0 {
		t.Fatalf("TotalWritten() = %d, want 0", got)
	}
}

func TestStreamBufferSmallWrites(t *testing.T) {
	b := newStreamBuffer(1024)
	n, err := b.Write([]byte("hello "))
	if err != nil || n != 6 {
		t.Fatalf("Write returned (%d, %v), want (6, nil)", n, err)
	}
	n, err = b.Write([]byte("world"))
	if err != nil || n != 5 {
		t.Fatalf("Write returned (%d, %v), want (5, nil)", n, err)
	}
	if got := b.String(); got != "hello world" {
		t.Fatalf("String() = %q", got)
	}
	if got := b.TotalWritten(); got != 11 {
		t.Fatalf("TotalWritten() = %d, want 11", got)
	}
}

func TestStreamBufferGrowsOnDemand(t *testing.T) {
	b := newStreamBuffer(4096)
	for i := 0; i < 8; i++ {
		if _, err := b.Write([]byte("0123456789abcdef")); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
	want := strings.Repeat("0123456789abcdef", 8)
	if got := b.String(); got != want {
		t.Fatalf("String() = %q, want full 128-byte content", got)
	}
	if got := b.TotalWritten(); got != 128 {
		t.Fatalf("TotalWritten() = %d, want 128", got)
	}

	// The backing array grows by doubling, so it stays within 2x the content
	// and never exceeds the limit.
	if got := len(b.data); got > 256 {
		t.Fatalf("backing array is %d bytes for 128 bytes of content, want at most 256", got)
	}
}

func TestStreamBufferSmallOutputStaysSmall(t *testing.T) {
	b := newStreamBuffer(10 << 20)
	if _, err := b.Write([]byte("tiny")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if got := len(b.data); got > 256 {
		t.Fatalf("backing array is %d bytes for a 4-byte write, want at most 256", got)
	}
}

func TestStreamBufferKeepsLastLimitBytes(t *testing.T) {
	b := newStreamBuffer(16)
	if _, err := b.Write([]byte("abcdefghijklmnopqrstuv")); err != nil { // 22 bytes
		t.Fatalf("Write failed: %v", err)
	}
	want := "ghijklmnopqrstuv" // last 16 bytes of the 22 written
	if got := b.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got := b.TotalWritten(); got != 22 {
		t.Fatalf("TotalWritten() = %d, want 22", got)
	}
}

func TestStreamBufferWrapAround(t *testing.T) {
	b := newStreamBuffer(16)
	payload := []byte("0123456789abcdef") // exactly limit
	for i := 0; i < 3; i++ {
		if _, err := b.Write(payload); err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}
	if got := b.String(); got != string(payload) {
		t.Fatalf("String() = %q, want %q", got, payload)
	}
	if got := b.TotalWritten(); got != 48 {
		t.Fatalf("TotalWritten() = %d, want 48", got)
	}

	// A small write that crosses the cursor boundary.
	if _, err := b.Write([]byte("XY")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	want := append(append([]byte{}, payload[2:]...), []byte("XY")...)
	if got := b.String(); got != string(want) {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStreamBufferLargeSingleWrite(t *testing.T) {
	b := newStreamBuffer(32)
	big := make([]byte, 100)
	for i := range big {
		big[i] = byte('a' + i%26)
	}
	n, err := b.Write(big)
	if err != nil || n != 100 {
		t.Fatalf("Write returned (%d, %v), want (100, nil)", n, err)
	}
	want := string(big[100-32:])
	if got := b.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got := b.TotalWritten(); got != 100 {
		t.Fatalf("TotalWritten() = %d, want 100", got)
	}
}

func TestStreamBufferZeroLimit(t *testing.T) {
	b := newStreamBuffer(0)
	if n, err := b.Write([]byte("x")); err != nil || n != 1 {
		t.Fatalf("Write returned (%d, %v), want (1, nil)", n, err)
	}
	if got := b.String(); got != "" {
		t.Fatalf("String() = %q, want empty", got)
	}
	if got := b.TotalWritten(); got != 1 {
		t.Fatalf("TotalWritten() = %d, want 1", got)
	}
}

// TestStreamBufferGrowthIsGeometric guards against reallocating on every write
// once the capacity gets close to the limit, which made growth quadratic.
func TestStreamBufferGrowthIsGeometric(t *testing.T) {
	const limit = 10 << 20
	chunk := make([]byte, 1024)
	b := newStreamBuffer(limit)

	reallocs, last := 0, 0
	for written := 0; written < 2*limit; written += len(chunk) {
		b.Write(chunk)
		if cap(b.data) != last {
			reallocs++
			last = cap(b.data)
		}
	}
	if reallocs > 20 {
		t.Fatalf("backing array was reallocated %d times, want logarithmic growth", reallocs)
	}
	if last != limit {
		t.Fatalf("final capacity = %d, want exactly the limit %d", last, limit)
	}
}

func TestStreamBufferSize(t *testing.T) {
	b := newStreamBuffer(1234)
	if got := b.Size(); got != 1234 {
		t.Fatalf("Size() = %d, want %d", got, 1234)
	}
}

// TestStreamBufferCrossingWrite is a regression test: when the buffer is only
// partially full and a write crosses the limit, the oldest bytes must be
// dropped (ring wraparound), not the newest ones.
func TestStreamBufferCrossingWrite(t *testing.T) {
	b := newStreamBuffer(10)
	if _, err := b.Write([]byte("0123456")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := b.Write([]byte("abcde")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	want := "23456abcde"
	if got := b.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got := b.TotalWritten(); got != 12 {
		t.Fatalf("TotalWritten() = %d, want 12", got)
	}

	// Keep writing past the limit: the ring must keep rotating.
	if _, err := b.Write([]byte("fghij")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if got := b.String(); got != "abcdefghij" {
		t.Fatalf("String() = %q, want %q", got, "abcdefghij")
	}
}

// TestStreamBufferEquivalentToCircbuf checks that streamBuffer behaves exactly
// like circbuf under random write sizes and a small limit, covering the cases
// (partially full buffer crossed by an oversized write) that fixed test tables
// tend to miss.
func TestStreamBufferEquivalentToCircbuf(t *testing.T) {
	const limit = 64

	for seed := int64(0); seed < 16; seed++ {
		rng := rand.New(rand.NewSource(seed))
		b := newStreamBuffer(limit)
		c, err := circbuf.NewBuffer(int64(limit))
		if err != nil {
			t.Fatalf("circbuf.NewBuffer failed: %v", err)
		}

		for i := 0; i < 512; i++ {
			n := rng.Intn(3*limit + 1) // from empty writes to single writes > limit
			buf := make([]byte, n)
			rng.Read(buf)

			if _, err := b.Write(buf); err != nil {
				t.Fatalf("seed %d, write %d: Write failed: %v", seed, i, err)
			}
			if _, err := c.Write(buf); err != nil {
				t.Fatalf("seed %d, write %d: circbuf Write failed: %v", seed, i, err)
			}

			want := c.Bytes()
			if got := b.Bytes(); string(got) != string(want) {
				t.Fatalf("seed %d, write %d (len %d): Bytes() = %q, want circbuf %q", seed, i, n, got, want)
			}
		}

		if got := b.TotalWritten(); got != c.TotalWritten() {
			t.Fatalf("seed %d: TotalWritten() = %d, want %d", seed, got, c.TotalWritten())
		}
	}
}
