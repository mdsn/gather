package source

import (
	"context"
	"io"
	"os"
	"slices"

	"github.com/mdsn/nexus/lib/watch"
)

type Manager struct {
	inotify *watch.Inotify
}

func NewManager() *Manager {
	ino, err := watch.NewInotify()
	if err != nil {
		return nil // XXX ??
	}
	return &Manager{inotify: ino}
}

func (m *Manager) AttachFile(ctx context.Context, spec *Spec) (*Source, error) {
	handle, err := m.inotify.Add(spec.Path)
	if err != nil {
		return nil, err
	}

	fp, err := os.Open(spec.Path)
	if err != nil {
		// XXX close handle.Out?
		return nil, err
	}

	src := &Source{
		Id:   spec.Id,
		Kind: KindFile,
		Done: make(chan struct{}),
		Out:  make(chan Output),
		Err:  make(chan error),
	}

	go func() {
		// Start at EOF
		offset, err := fileSize(fp)
		if err != nil {
			return // XXX ?
		}

		truncating := false
		partial := NewFixedBuffer(4096 * 2)
		buf := make([]byte, 4096)
		for _ = range handle.Out {
			sz, err := fileSize(fp)
			if err != nil {
				return
			}

			// File was truncated, bring offset back
			if sz < offset {
				offset = sz
				continue
			}

			// read file from offset
			_, err = fp.Seek(offset, 0)
			if err != nil {
				return
			}

			for {
				buf = buf[:cap(buf)]

				n, err := fp.Read(buf)
				if n == 0 && err == io.EOF {
					break
				}

				if err != nil {
					return
				}

				// Consume all newlines read
				nlix := slices.Index(buf, '\n')
				for nlix != -1 {
					if truncating {
						truncating = false
					} else {
						partial.Write(buf[:nlix])
						output(partial, src)
					}
					// Trim prefix regardless of the fixed buffer being filled.
					buf = discardPrefix(buf, nlix+1)
					nlix = slices.Index(buf, '\n')
				}

				// Here buf is either the suffix after the final newline, or a
				// fully read buffer that never had a newline to begin with.
				// Same thing. If truncating, discard the entire buffer.
				if !truncating {
					// Write last chunk to fixed buffer; output only if it
					// fills up.
					_, err := partial.Write(buf)
					if err != nil {
						truncating = true
						output(partial, src)
					}
				}

				offset += int64(n)
			}
		}
	}()

	return src, nil
}

func output(b *FixedBuffer, src *Source) {
	src.Send(b.buf)
	b.Clear()
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

func fileSize(fp *os.File) (int64, error) {
	stat, err := fp.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

type ErrFull struct{}

func (e *ErrFull) Error() string {
	return "ring buffer full"
}

// A fixed size buffer that signals full.
type FixedBuffer struct {
	buf []byte
}

func NewFixedBuffer(n int) *FixedBuffer {
	return &FixedBuffer{buf: make([]byte, n)}
}

func (b *FixedBuffer) Write(p []byte) (int, error) {
	// How many bytes can we copy into the buffer
	avail := cap(b.buf) - len(b.buf)
	// Append up to that number of bytes
	b.buf = append(b.buf, p[:avail]...)
	// If the buffer was filled we used all the available space
	if len(b.buf) == cap(b.buf) {
		return avail, &ErrFull{}
	}
	// If the buffer was not filled we copied all the source bytes
	return len(p), nil
}

func (b *FixedBuffer) Clear() {
	b.buf = b.buf[:0]
}
