package source

import (
	"context"

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
	// When attaching a new file, Inotify.Add() it.
	// Events start pouring in on the Ino goroutine.
	// They get fanned out to each Watch out channel.
	// Here on AttachFile we listen on a specific Watch channel,
	// and write out into the Source.
	return nil, nil
}
