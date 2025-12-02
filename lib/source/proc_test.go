package source

import (
	"testing"
	"time"
)

func consume(src *Source, outC chan []byte) {
	var out []byte
	for msg := range src.Out {
		out = append(out, msg.Bytes...)
	}
	outC <- out
}

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

	// TODO check output
}

func TestAttachProc_FailsCommandNotFound(t *testing.T) {
	ctx := t.Context()
	spec := NewSpec("nonexistent", "berengario", []string{})
	src, err := attachProc(ctx, spec)

	if err == nil {
		t.Fatalf("err is nil")
	}
	if src != nil {
		t.Fatalf("src not nil: %v", src)
	}
}

func TestAttachProc_RedirectsStdout(t *testing.T) {
	ctx := t.Context()
	const want = "yes! radiant lyre speak to me become a voice"
	spec := NewSpec("sappho", "echo", []string{"-n", want})

	src, err := attachProc(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go consume(src, outC)

	<-src.Done

	out := <-outC
	if string(out) != want {
		t.Fatalf("stdout = '%s', want '%s'", out, want)
	}
}

func TestAttachProc_StreamsMultipleLines(t *testing.T) {
}

func TestAttachProc_TruncatesLongLine(t *testing.T) {
}

func TestAttachProc_LastLineNoNewline(t *testing.T) {
}

func TestAttachProc_CancelContext(t *testing.T) {
}

func TestAttachProc_ExternalSigterm(t *testing.T) {
}

func TestAttachProc_ChanClosedOnExit(t *testing.T) {
}

func TestAttachProc_MultipleSources(t *testing.T) {
}

func TestAttachProc_ChildKeepsPipeOpen(t *testing.T) {
}
