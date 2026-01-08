package manager

import (
	"context"
	"errors"

	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/source/file"
	"github.com/mdsn/nexus/lib/source/proc"
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

func (m *Manager) Attach(ctx context.Context, spec *source.Spec) (*source.Source, error) {
	switch spec.Kind {
	case source.KindProc:
		return proc.Attach(ctx, spec)
	case source.KindFile:
		handle, err := m.inotify.Add(spec.Path)
		if err != nil {
			return nil, err
		}
		return file.Attach(ctx, spec, handle)
	}
	return nil, errors.New("unknown SourceKind")
}
