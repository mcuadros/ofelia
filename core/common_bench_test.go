package core

import (
	"testing"

	"github.com/armon/circbuf"
)

// BenchmarkNewExecution measures the memory cost of creating an Execution.
// With lazy stream buffers, a fresh execution allocates only a few bytes; the
// 10 MiB stream limit is reserved on demand as output is written.
func BenchmarkNewExecution(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := NewExecution()
		if e.OutputStream == nil || e.ErrorStream == nil {
			b.Fatal("expected non-nil streams")
		}
	}
}

// TestNewExecutionLazyAllocation documents that NewExecution no longer
// eagerly reserves the full stream limit: a fresh execution allocates far less
// than two 10 MiB buffers, and memory grows only as output is written.
func TestNewExecutionLazyAllocation(t *testing.T) {
	const perStreamLimit = 10 * 1024 * 1024

	e := NewExecution()
	if got := e.OutputStream.Size(); got != perStreamLimit {
		t.Fatalf("OutputStream size = %d, want %d", got, perStreamLimit)
	}
	if got := e.ErrorStream.Size(); got != perStreamLimit {
		t.Fatalf("ErrorStream size = %d, want %d", got, perStreamLimit)
	}

	var sink *Execution
	allocs := testing.AllocsPerRun(10, func() { sink = NewExecution() })
	_ = sink
	if allocs > 10 {
		t.Fatalf("NewExecution made %v allocations, want only the Execution and its small buffers", allocs)
	}
	if len(e.OutputStream.data) != 0 || len(e.ErrorStream.data) != 0 {
		t.Fatalf("fresh streams reserved %d/%d bytes, want 0", len(e.OutputStream.data), len(e.ErrorStream.data))
	}
}

// BenchmarkStreamWrite compares sustained throughput for a job that produces
// far more output than the limit (64 MiB in 32 KiB chunks against 10 MiB), so
// the buffer grows to its limit and then wraps around.
func BenchmarkStreamWrite(b *testing.B) {
	chunk := make([]byte, 32*1024)
	const total = 64 << 20

	b.Run("streamBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(total)
		for i := 0; i < b.N; i++ {
			s := newStreamBuffer(maxStreamSize)
			for w := 0; w < total/len(chunk); w++ {
				s.Write(chunk)
			}
		}
	})
	b.Run("circbuf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(total)
		for i := 0; i < b.N; i++ {
			c, _ := circbuf.NewBuffer(maxStreamSize)
			for w := 0; w < total/len(chunk); w++ {
				c.Write(chunk)
			}
		}
	})
}
