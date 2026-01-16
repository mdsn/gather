package source

import (
	"context"
	"time"
)

const (
	MaxLineLength = 4096
)

type SourceKind uint8

const (
	KindFile SourceKind = iota
	KindProc
)

type Output struct {
	Id         string
	CapturedAt time.Time
	Bytes      []byte
}

type Source struct {
	Id   string
	Kind SourceKind
	// Closed when reading goroutine exits.
	Done chan struct{}
	// Closed when reading goroutine is ready to read.
	Ready chan struct{}
	// Output is sent on this channel.
	Out chan Output
	Err chan error
	// Terminates the execution of this source.
	Cancel context.CancelFunc
}

func (src *Source) Send(b []byte) {
	buf := make([]byte, len(b))
	copy(buf, b)
	src.Out <- Output{
		Id:         src.Id,
		CapturedAt: time.Now(),
		Bytes:      buf,
	}
}

type Spec struct {
	Id   string
	Kind SourceKind
	Path string
	Args []string
}
