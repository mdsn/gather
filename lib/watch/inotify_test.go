package watch

import (
	"testing"
	"os"
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
	defer os.Remove(tmp.Name())
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	_, err = ino.Add(tmp.Name())
	if err != nil {
		t.Fatalf("add err: %v", err)
	}

	if len(ino.wds) != 1 {
		t.Fatalf("wrong number of watch descriptors")
	}
}


