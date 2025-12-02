package source

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func Consume(src *Source, outC chan []byte) {
	var out []byte
	for msg := range src.Out {
		out = append(out, msg.Bytes...)
	}
	outC <- out
}

func Map[T any, R any](xs []T, f func(T) R) []R {
	var ys []R
	for _, x := range xs {
		ys = append(ys, f(x))
	}
	return ys
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
	go Consume(src, outC)

	<-src.Done

	out := <-outC
	if string(out) != want {
		t.Fatalf("stdout = '%s', want '%s'", out, want)
	}
}

func TestAttachProc_StreamsMultipleLines(t *testing.T) {
	ctx := t.Context()
	lines := []string{
		"O God, what great kindness have we done in times past and forgotten it,",
		"That thou givest this wonder unto us,",
		"O God of waters?",
	}
	cmd := fmt.Sprintf("echo '%s'; echo '%s'; echo '%s'", lines[0], lines[1], lines[2])
	spec := NewSpec("pound", "sh", []string{"-c", cmd})

	src, err := attachProc(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go Consume(src, outC)

	out := <-outC
	outLines := Map(
		slices.Collect(strings.Lines(string(out))),
		func(s string) string { return strings.TrimRight(s, "\n") },
	)

	if !slices.Equal(lines, outLines) {
		t.Fatalf("outLines: %q, want: %q", outLines, lines)
	}
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
