package watch

import (
	"os"
	"testing"
	"time"
)

func TestNewInotify_HasFd(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}
	if ino.ifd == 0 {
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

	select {
	case <-ino.done:
	case <-time.After(time.Second):
		t.Fatal("timeout expired")
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

func TestInotify_ReceiveEvent(t *testing.T) {
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

	bytes := []byte("The show is not the show, but they that go.")
	if _, err := tmp.Write(bytes); err != nil {
		t.Fatalf("write error: %v", err)
	}

	ev := <-handle.Out
	if int(ev.Wd) != handle.wd {
		t.Fatal("ev.Wd != handle.wd:", ev.Wd, "!=", handle.wd)
	}
}

func TestInotify_ReceiveEventDifferentFiles(t *testing.T) {
	ino, err := NewInotify()
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	tmp1, err := os.CreateTemp("", "inotest")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	tmp2, err := os.CreateTemp("", "inotest")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	defer os.Remove(tmp1.Name())
	defer os.Remove(tmp2.Name())

	handle1, err := ino.Add(tmp1.Name())
	if err != nil {
		t.Fatalf("add err: %v", err)
	}

	handle2, err := ino.Add(tmp2.Name())
	if err != nil {
		t.Fatalf("add err: %v", err)
	}

	bytes1 := []byte("The show is not the show, but they that go.")
	if _, err := tmp1.Write(bytes1); err != nil {
		t.Fatalf("write error: %v", err)
	}

	bytes2 := []byte("The trouble's small, the fun is great.")
	if _, err := tmp2.Write(bytes2); err != nil {
		t.Fatalf("write error: %v", err)
	}

	ev1 := <-handle1.Out
	ev2 := <-handle2.Out

	if int(ev1.Wd) != handle1.wd {
		t.Fatal("Unexpected wd: got", ev1.Wd, "wanted", handle1.wd)
	}

	if int(ev2.Wd) != handle2.wd {
		t.Fatal("Unexpected wd: got", ev2.Wd, "wanted", handle2.wd)
	}

	ino.Rm(handle1)
	ino.Rm(handle2)
}
