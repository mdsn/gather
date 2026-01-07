package source

import (
	"context"
	"os"
	"testing"
	"time"
)

func MakeSpec(id string) (*os.File, *Spec, error) {
	tmp, err := os.CreateTemp("", "filetest")
	if err != nil {
		return nil, nil, err
	}
	return tmp, &Spec{Id: id, Kind: KindFile, Path: tmp.Name()}, nil
}

func CollectLines(srcC chan Output) [][]byte {
	lines := make([][]byte, 0)
	for output := range srcC {
		lines = append(lines, output.Bytes)
	}
	return lines
}

func consume(src *Source, outC chan [][]byte) {
	lines := CollectLines(src.Out)
	outC <- lines
}

func TestAttachFile_OutputLines(t *testing.T) {
	m := NewManager()
	defer m.inotify.Close()

	tmp, spec, err := MakeSpec("output-lines")
	if err != nil {
		t.Fatalf("MakeSpec: %v", err)
	}
	defer os.Remove(spec.Path)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	src, err := m.AttachFile(ctx, spec)
	if err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	lineC := make(chan []byte, 16)
	go func() {
		defer close(lineC)
		for out := range src.Out {
			select {
			case lineC <- out.Bytes:
			case <-ctx.Done():
				return
			}
		}
	}()

	<-src.Ready

	bytes := []byte("Is't life, I ask, is't even prudence,\nTo bore thyself and bore the students?\n")
	if _, err := tmp.Write(bytes); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()

	var lines [][]byte
	for len(lines) < 2 {
		select {
		case line := <-lineC:
			lines = append(lines, line)
		case <-deadline.C:
			t.Fatalf("timeout; wanted 2 lines, got %d", len(lines))
		}
	}

	cancel()

	select {
	case <-src.Done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for src.Done")
	}

	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}
}
