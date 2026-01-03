package watch

import (
	"os"
	"testing"
)

func TestNewInotify_HasFd(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}
	if ino.fd == 0 {
		t.Fatalf("fd is 0")
	}
}

func TestInotifyClose_Success(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}
	err = ino.Close()
	if err != nil {
		t.Fatalf("err closing inotify: %v", err)
	}
}

func TestInotifyAdd_WatchCreated(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	tmp, err := os.CreateTemp("", "inotest")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())

	_, err = ino.Add(tmp.Name())
	if err != nil {
		t.Fatalf("add err: %v", err)
	}

	if len(ino.wds) != 1 {
		t.Fatalf("wrong number of watch descriptors")
	}
}

func TestInotifyRm_ChanClosed(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	tmp, err := os.CreateTemp("", "inotest")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())

	handle, err := ino.Add(tmp.Name())
	if err != nil {
		t.Fatalf("add err: %v", err)
	}

	err = ino.Rm(handle)
	if err != nil {
		t.Fatalf("rm err: %v", err)
	}

	if _, ok := <-handle.Out; ok {
		t.Fatalf("out channel not closed after Rm")
	}

	if _, ok := ino.wds[handle.wd]; ok {
		t.Fatalf("Watch still in map after Rm")
	}
}
