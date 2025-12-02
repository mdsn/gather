package source

import (
	"testing"
	"time"
)

func NewSpec(id, cmd string, args []string) *Spec {
	return &Spec{Id: id, Kind: KindProc, Path: cmd, Args: args}
}

func TestAttachProc_NoOutput(t *testing.T) {
	ctx := t.Context()
	spec := NewSpec("true", "true", []string{})

	src, err := attachProc(ctx, spec)

	if err != nil {
		t.Fatalf("err not nil: %v", err)
	}
	if src == nil {
		t.Fatalf("src is nil")
	}
	if src.Id != "true" {
		t.Fatal("Id =", src.Id, "wanted true")
	}
	if src.Kind != KindProc {
		t.Fatal("Kind =", src.Kind, "wanted", KindProc)
	}

	select {
	case <-src.Done:
	case <-time.After(time.Second):
		t.Fatalf("timeout expired")
	}
}
