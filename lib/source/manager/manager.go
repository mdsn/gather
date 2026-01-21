package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mdsn/gather/lib/source"
	"github.com/mdsn/gather/lib/source/file"
	"github.com/mdsn/gather/lib/source/proc"
	"github.com/mdsn/gather/lib/watch"
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
		Events:  make(chan source.Output),
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
			return fmt.Errorf("inotify: %v", err)
		}

		src, err = file.Attach(ctx, spec, handle)
		if err != nil {
			err = errors.Join(err, m.inotify.Rm(handle))
			return fmt.Errorf("attach: %v", err)
		}
	default:
		return errors.New("unknown SourceKind")
	}

	m.mu.Lock()
	m.sources[src.Id] = src
	m.mu.Unlock()

	// Fan into the manager's Events channel.
	go func() {
		// Remove source, ignoring error
		defer m.Remove(src.Id)

		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-src.Out:
				if !ok {
					return
				}
				select { // Prevent blocking on the send
				case m.Events <- out:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return nil
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	src, ok := m.sources[id]
	if !ok {
		m.mu.Unlock()
		return errors.New(fmt.Sprintf("source '%s' not found", id))
	}

	delete(m.sources, id)
	m.mu.Unlock()

	src.Cancel()
	<-src.Done
	return nil
}
