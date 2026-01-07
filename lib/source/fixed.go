package source

type ErrFull struct{}

func (e *ErrFull) Error() string {
	return "fixed buffer full"
}

// A fixed size buffer that signals when it is full.
type FixedBuffer struct {
	buf []byte
}

func NewFixedBuffer(n int) *FixedBuffer {
	return &FixedBuffer{buf: make([]byte, 0, n)}
}

func (b *FixedBuffer) Len() int {
	return len(b.buf)
}

func (b *FixedBuffer) Write(p []byte) (int, error) {
	// How many bytes can we copy into the buffer
	n := min(cap(b.buf)-len(b.buf), len(p))
	// Append up to that number of bytes
	b.buf = append(b.buf, p[:n]...)
	// If the buffer was filled we used all the available space
	if len(b.buf) == cap(b.buf) {
		return n, &ErrFull{}
	}
	// If the buffer was not filled we copied all the source bytes
	return len(p), nil
}

func (b *FixedBuffer) Clear() {
	b.buf = b.buf[:0]
}
