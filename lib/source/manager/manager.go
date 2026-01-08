package manager

import (
	"context"
	"os"

	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/source/file"
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

	go file.Tail(ctx, src, fp, handle.Out)

	return src, nil
}
