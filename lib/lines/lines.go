package lines

import (
	"iter"
	"slices"
)

// A buffer to turn byte-oriented input into line-oriented output.
//
// The idea is that the user will call Add() with the result of a read(2) call,
// then repeatedly call Line() to pull lines of output from the LineBuffer
// until there aren't any. The LineBuffer will take care of truncating lines
// across read(2) calls, parsing multiple lines out of individual reads, and
// maintaining line prefixes across calls.
type LineBuffer struct {
	fb         *FixedBuffer
	truncating bool
	buf        []byte
}

func NewLineBuffer(n int) *LineBuffer {
	return &LineBuffer{fb: NewFixedBuffer(n)}
}

// LineBuffer takes ownership of slice b until all lines yielded by
// Lines() have been exhausted.
func (lb *LineBuffer) Add(b []byte) {
	lb.buf = b
}

// Yield lines from the buffer given to Add(). The yielded lines are
// cloned; ownership is with the caller.
func (lb *LineBuffer) Lines() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		nl := slices.Index(lb.buf, '\n')
		for nl != -1 {
			if lb.truncating {
				lb.truncating = false
			} else {
				lb.fb.Write(lb.buf[:nl])
				line := lb.take()
				if !yield(line) {
					return // ??
				}
			}
			lb.buf = discardPrefix(lb.buf, nl+1)
			nl = slices.Index(lb.buf, '\n')
		}

		if !lb.truncating {
			_, err := lb.fb.Write(lb.buf)
			if err != nil {
				lb.truncating = true
				if !yield(lb.take()) {
					return // ??
				}
			}
		}
	}
}

// Clone the content of the accumulated buffer into a new slice and
// clear it.
func (lb *LineBuffer) take() []byte {
	line := make([]byte, lb.fb.Len())
	copy(line, lb.fb.buf)
	lb.fb.Clear()
	return line
}

// Discard up to and not including index n; if n > len(b), discard the whole
// thing.
func discardPrefix(b []byte, n int) []byte {
	if len(b) < n {
		b = b[:0]
	} else {
		b = b[n:]
	}
	return b
}
