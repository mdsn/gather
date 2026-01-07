package source

import (
	"context"
	"io"
	"os"

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

		lb := NewLineBuffer(4096 * 2)
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
