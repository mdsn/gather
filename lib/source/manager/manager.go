package manager

import (
	"context"
	"errors"
	"sync"

	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/source/file"
	"github.com/mdsn/nexus/lib/source/proc"
	"github.com/mdsn/nexus/lib/watch"
)

type Manager struct {
	inotify *watch.Inotify
	// Synchronizes access to sources
	mu      sync.Mutex
	sources map[string]*source.Source
	// Output from sources is fanned into this channel
	Events chan source.Output
}

func NewManager() *Manager {
	ino, err := watch.NewInotify()
	if err != nil {
		return nil // XXX ??
	}
	return &Manager{
		inotify: ino,
		sources: make(map[string]*source.Source),
	}
}

func (m *Manager) Close() error {
	return m.inotify.Close()
}

func (m *Manager) Attach(ctx context.Context, spec *source.Spec) error {
	var src *source.Source
	var err error

	switch spec.Kind {
	case source.KindProc:
		src, err = proc.Attach(ctx, spec)
		if err != nil {
			return err
		}

	case source.KindFile:
		handle, err := m.inotify.Add(spec.Path)
		if err != nil {
			return err
		}

		src, err = file.Attach(ctx, spec, handle)
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown SourceKind")
	}

	m.mu.Lock()
	m.sources[src.Id] = src
	m.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case out := <-src.Out:
				m.Events <- out
			}
		}
	}()

	return nil
}
