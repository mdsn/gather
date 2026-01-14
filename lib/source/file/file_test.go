package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/watch"
)

func MakeSpec(id string) (*os.File, *source.Spec, error) {
	tmp, err := os.CreateTemp("", "filetest")
	if err != nil {
		return nil, nil, err
	}
	return tmp, &source.Spec{Id: id, Kind: source.KindFile, Path: tmp.Name()}, nil
}

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

// Collect a specific number of lines from a line channel with a timeout
func collect(n int, lineC chan []byte, deadline time.Duration) ([][]byte, error) {
	timeout := time.NewTimer(deadline)
	defer timeout.Stop()

	var lines [][]byte
	for len(lines) < n {
		select {
		case line := <-lineC:
			lines = append(lines, line)
		case <-timeout.C:
			return nil, errors.New(fmt.Sprintf("timeout; wanted %d lines, got %d", n, len(lines)))
		}
	}

	return lines, nil
}

// Write some output into the given file and sync it.
func write(f *os.File, b []byte) (n int, err error) {
	if n, err = f.Write(b); err != nil {
		return -1, errors.New(fmt.Sprintf("write: %v", err))
	}
	if err = f.Sync(); err != nil {
		return -1, errors.New(fmt.Sprintf("sync: %v", err))
	}
	return n, nil
}

// Wait for src.Done with the given timeout
func wait(src *source.Source, deadline time.Duration) error {
	select {
	case <-src.Done:
	case <-time.After(deadline):
		return errors.New("timeout waiting for src.Done")
	}
	return nil
}

func TestAttachFile_OutputLines(t *testing.T) {
	ino, err := watch.NewInotify()
	if err != nil {
		t.Fatalf("Inotify: %v", err)
	}
	defer ino.Close()

	tmp, spec, err := MakeSpec("output-lines")
	if err != nil {
		t.Fatalf("MakeSpec: %v", err)
	}
	defer os.Remove(spec.Path)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	handle, err := ino.Add(spec.Path)
	if err != nil {
		t.Fatalf("inotify Add: %v", err)
	}

	src, err := Attach(ctx, spec, handle)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	lineC := make(chan []byte, 16)
	go consume(ctx, src, lineC)
	<-src.Ready

	line1 := "Is't life, I ask, is't even prudence,"
	line2 := "To bore thyself and bore the students?"
	bytes := []byte(fmt.Sprintf("%s\n%s\n", line1, line2))
	_, err = write(tmp, bytes)
	if err != nil {
		t.Fatal(err)
	}

	lines, err := collect(2, lineC, time.Second)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	cancel()

	err = wait(src, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}
	if string(lines[0]) != line1 || string(lines[1]) != line2 {
		t.Fatalf("unexpected output: '%s' '%s'", line1, line2)
	}
}

func TestAttachFile_TruncateFile(t *testing.T) {
	t.Skip("non-deterministic behavior; see 03-truncate.md")

	ino, err := watch.NewInotify()
	if err != nil {
		t.Fatalf("Inotify: %v", err)
	}
	defer ino.Close()

	tmp, spec, err := MakeSpec("truncate")
	if err != nil {
		t.Fatalf("MakeSpec: %v", err)
	}
	defer os.Remove(spec.Path)

	// Write and sync before attaching to the file to simulate previous content.
	line1 := "To those who gaze on thee what language could they speak?" // len 57
	_, err = write(tmp, []byte(line1))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	handle, err := ino.Add(spec.Path)
	if err != nil {
		t.Fatalf("inotify Add: %v", err)
	}

	src, err := Attach(ctx, spec, handle)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	lineC := make(chan []byte, 16)
	go consume(ctx, src, lineC)
	<-src.Ready

	// Truncate file
	err = tmp.Truncate(0)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// XXX truncate(2) delivers IN_MODIFY; if tail() is too slow to read it out
	// of the event stream and the write below happens, the kernel coalesces
	// both IN_MODIFY into 1, it never updates the offset, and reads from 57 to
	// 64. We somehow need a different approach or a different guarantee.

	// truncate(2) does not change the file offset. Simulate a process opening
	// the file after truncation by seeking to 0.
	_, err = tmp.Seek(0, 0)
	if err != nil {
		t.Fatalf("seek: %v", err)
	}

	// Write another line. It comes up in `lines`.
	line2 := "You can lead a horse to water, but you can't make him backstroke" // len 64
	_, err = write(tmp, []byte(fmt.Sprintf("%s\n", line2)))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	lines, err := collect(1, lineC, time.Second)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	cancel()

	err = wait(src, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 1 {
		t.Fatal("wrong length, want 1, got", len(lines))
	}
	if string(lines[0]) != line2 {
		t.Fatalf("unexpected output: '%s'", lines[0])
	}

	size, err := fileSize(tmp)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if int(size) != len(line2) {
		t.Fatal("wrong file size, want", len(line2), "got", size)
	}
}

// TODO test truncate past EOF (hole in file)
// TODO test multiple files/sources
// TODO test ino.Add -> rm file -> Attach (race situation)
// TODO test line truncation
// TODO test lingering buffered partial line
