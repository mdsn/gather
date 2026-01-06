package source

import (
	"context"
	"errors"
	"time"
)

const (
	maxLineLength = 4096
)

type SourceKind uint8

const (
	KindFile SourceKind = iota
	KindProc
)

type Output struct {
	CapturedAt time.Time
	Bytes      []byte
}

type Source struct {
	Id   string
	Kind SourceKind
	Done chan struct{}
	Out  chan Output
	Err  chan error
}

func (src *Source) Send(b []byte) {
	buf := make([]byte, len(b))
	copy(buf, b)
	src.Out <- Output{
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

func Attach(ctx context.Context, spec *Spec) (*Source, error) {
	switch spec.Kind {
	case KindFile:
		return attachFile(ctx, spec)
	case KindProc:
		return attachProc(ctx, spec)
	default:
		return nil, errors.New("unknown source kind")
	}
}

func attachFile(ctx context.Context, spec *Spec) (*Source, error) {
	return nil, nil
}
