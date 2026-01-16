package proc

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/mdsn/nexus/lib/source"
)

// Consume lines from a source's Out channel into a line channel
func consume(ctx context.Context, src *source.Source, lineC chan []byte) {
	defer close(lineC)
	for out := range src.Out {
		select {
		case lineC <- out.Bytes:
		case <-ctx.Done():
			return
		}
	}
}

// Collect lines from a line channel with a timeout
func collect(lineC chan []byte, deadline time.Duration) [][]byte {
	timeout := time.NewTimer(deadline)
	defer timeout.Stop()

	var lines [][]byte
	for {
		select {
		case line, ok := <-lineC:
			if !ok {
				return lines
			}
			lines = append(lines, line)
		case <-timeout.C:
			return lines
		}
	}

	return lines
}

func Map[T any, R any](xs []T, f func(T) R) []R {
	var ys []R
	for _, x := range xs {
		ys = append(ys, f(x))
	}
	return ys
}

func NewSpec(id, cmd string, args []string) *source.Spec {
	return &source.Spec{Id: id, Kind: source.KindProc, Path: cmd, Args: args}
}

func TestAttachProc_NoOutput(t *testing.T) {
	ctx := t.Context()
	spec := NewSpec("true", "true", []string{})

	src, err := Attach(ctx, spec)

	outC := make(chan []byte)
	go consume(ctx, src, outC)
	out := <-outC

	if err != nil {
		t.Fatalf("err not nil: %v", err)
	}
	if src == nil {
		t.Fatalf("src is nil")
	}
	if src.Id != "true" {
		t.Fatal("Id =", src.Id, "wanted true")
	}
	if src.Kind != source.KindProc {
		t.Fatal("Kind =", src.Kind, "wanted", source.KindProc)
	}

	select {
	case <-src.Done:
	case <-time.After(time.Second):
		t.Fatalf("timeout expired")
	}

	if string(out) != "" {
		t.Fatalf("unexpected output: %q", string(out))
	}

	if _, ok := <-src.Out; ok {
		t.Fatalf("out channel not closed on exit")
	}
}

func TestAttachProc_FailsCommandNotFound(t *testing.T) {
	ctx := t.Context()
	spec := NewSpec("nonexistent", "berengario", []string{})
	src, err := Attach(ctx, spec)

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

	src, err := Attach(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go consume(ctx, src, outC)

	<-src.Done

	out := <-outC
	if string(out) != want {
		t.Fatalf("stdout = '%s', want '%s'", out, want)
	}
}

func TestAttachProc_StreamsMultipleLines(t *testing.T) {
	ctx := t.Context()
	lines := [][]byte{
		[]byte("O God, what great kindness have we done in times past and forgotten it,"),
		[]byte("That thou givest this wonder unto us,"),
		[]byte("O God of waters?"),
	}
	cmd := fmt.Sprintf("echo '%s'; echo '%s'; echo '%s'", string(lines[0]), string(lines[1]), string(lines[2]))
	spec := NewSpec("pound", "sh", []string{"-c", cmd})

	src, err := Attach(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go consume(ctx, src, outC)

	out := collect(outC, time.Second)

	if len(lines) != len(out) {
		t.Fatalf("out: %q, want: %q", out, lines)
	}

	for i := range len(out) {
		if !slices.Equal(out[i], lines[i]) {
			t.Fatalf("out: %q, want: %q", out, lines)
		}
	}
}

func TestAttachProc_TruncatesLongLine(t *testing.T) {
	ctx := t.Context()
	// As of today, bufio.Reader defaultBufSize is 4096. Build a larger string
	// than that and see that it is truncated.
	wantLen := source.MaxLineLength
	strLen := 2 * wantLen
	alphabet := "a b c u v w x y z ' '"
	// No newline
	cmd := fmt.Sprintf("echo -n \"$(shuf -er -n %d %s | tr -d '\\n')\"", strLen, alphabet)
	spec := NewSpec("long", "sh", []string{"-c", cmd})

	src, err := Attach(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go consume(ctx, src, outC)

	out := <-outC
	outStr := string(out)
	if len(outStr) != wantLen {
		t.Fatalf("len(outStr) = %d, want: %d\noutStr: '%s'", len(outStr), wantLen, outStr)
	}
}

func TestAttachProc_LastLineNoNewline(t *testing.T) {
	ctx := t.Context()
	want := [][]byte{[]byte("abc"), []byte("def"), []byte("ghi")}
	spec := NewSpec("newline", "sh", []string{"-c", "echo abc; echo def; echo -n ghi"})

	src, err := Attach(ctx, spec)
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	outC := make(chan []byte)
	go consume(ctx, src, outC)

	lines := collect(outC, time.Second)

	if len(lines) != len(want) {
		t.Fatalf("lines: %q; want: %q", lines, want)
	}

	for i := range len(lines) {
		if !slices.Equal(lines[i], want[i]) {
			t.Fatalf("lines: %q; want: %q", lines, want)
		}
	}
}

func TestAttachProc_CancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	spec := NewSpec("terminate", "sh", []string{"-c", "sleep 10; echo hullaballoo"})

	src, _ := Attach(ctx, spec)

	outC := make(chan []byte)
	go consume(ctx, src, outC)

	cancel()

	select {
	case <-src.Done:
	case <-time.After(time.Second):
		t.Fatalf("timeout expired")
	}

	out := <-outC
	if string(out) != "" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestAttachProc_MultipleSources(t *testing.T) {
	ctx := t.Context()
	const want = 6
	spec1 := NewSpec("fst", "sh", []string{"-c", "echo one; echo two; echo three"})
	spec2 := NewSpec("snd", "sh", []string{"-c", "echo 111; echo 222; echo 33333"})

	src1, _ := Attach(ctx, spec1)
	src2, _ := Attach(ctx, spec2)

	outC := make(chan source.Output)
	consume := func(procOut chan source.Output) {
		for msg := range procOut {
			outC <- msg
		}
	}

	var wg sync.WaitGroup
	wg.Go(func() { consume(src1.Out) })
	wg.Go(func() { consume(src2.Out) })

	go func() {
		wg.Wait()
		close(outC)
	}()

	var msgs []source.Output
	for msg := range outC {
		msgs = append(msgs, msg)
	}

	<-src1.Done
	<-src2.Done

	if len(msgs) != want {
		t.Fatalf("wrong number of messages: %d, want %d", len(msgs), want)
	}
}

func TestAttachProc_ChildKeepsPipeOpen(t *testing.T) {
	ctx := t.Context()
	streamGracePeriod = 50 * time.Millisecond
	// sh spawns sleep in the background then exits; sleep inherits the write
	// end of the pipe and keeps it open
	spec := NewSpec("fork", "sh", []string{"-c", "sleep 1000 &"})

	src, _ := Attach(ctx, spec)

	select {
	case <-src.Done:
	case <-time.After(time.Second):
		t.Fatalf("timeout expired")
	}
}
