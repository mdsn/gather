package manager

import (
	"context"
	"io"
	"os"

	"github.com/mdsn/nexus/lib/lines"
	"github.com/mdsn/nexus/lib/source"
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

func (m *Manager) Close() error {
	return m.inotify.Close()
}

func (m *Manager) AttachFile(ctx context.Context, spec *source.Spec) (*source.Source, error) {
	handle, err := m.inotify.Add(spec.Path)
	if err != nil {
		return nil, err
	}

	fp, err := os.Open(spec.Path)
	if err != nil {
		// XXX close handle.Out?
		return nil, err
	}

	// TODO source.NewSource(...)
	src := &source.Source{
		Id:    spec.Id,
		Kind:  source.KindFile,
		Done:  make(chan struct{}),
		Ready: make(chan struct{}),
		Out:   make(chan source.Output),
		Err:   make(chan error),
	}

	go func() {
		defer close(src.Done)

		// Start at EOF
		offset, err := fileSize(fp)
		if err != nil {
			return // XXX ?
		}

		lb := lines.NewLineBuffer(4096 * 2)
		buf := make([]byte, 4096)

		// Start listening
		close(src.Ready)

	outer:
		for {
			select {
			case <-ctx.Done():
				break outer
			case <-handle.Out:
				// Continue
			}

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

				offset += int64(n)

				lb.Add(buf)
				for line := range lb.Lines() {
					src.Send(line)
				}
			}
		}
	}()

	return src, nil
}

func fileSize(fp *os.File) (int64, error) {
	stat, err := fp.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}
