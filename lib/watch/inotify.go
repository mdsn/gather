package watch

import (
	"errors"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	InotifyBufferSize = 4096
	// TODO IN_IGNORED can also happen when the os clears a watch due to delete
	// or unmount. Remove watch when that happens.
	InotifyMask = syscall.IN_MODIFY
)

type Event struct {
	Wd     int32
	Cookie uint32
	Mask   uint32
	Name   string
}

type Watch struct {
	path   string
	offset int
	out    chan Event
}

type WatchHandle struct {
	wd  int
	Out chan Event
}

type Inotify struct {
	mu  sync.Mutex
	fd  int
	wds map[int]*Watch
}

func NewInotify() (*Inotify, error) {
	// XXX do we need FD_CLOEXEC?
	fd, err := syscall.InotifyInit()
	if err != nil {
		// XXX test all errno for this syscall and wrap
		return nil, err
	}

	ino := &Inotify{
		fd:  fd,
		wds: make(map[int]*Watch),
	}
	go inotifyReceive(ino)

	return ino, nil
}

func (ino *Inotify) Close() error {
	err := syscall.Close(ino.fd)
	if err != nil {
		return err // ??
	}
	return nil
}

func (ino *Inotify) Add(path string) (*WatchHandle, error) {
	wd, err := syscall.InotifyAddWatch(ino.fd, path, InotifyMask)
	if err != nil {
		return nil, err
	}
	outC := make(chan Event)

	ino.mu.Lock()
	ino.wds[wd] = &Watch{
		path: path,
		out:  outC,
	}
	ino.mu.Unlock()

	return &WatchHandle{wd: wd, Out: outC}, nil
}

func (ino *Inotify) Rm(handle *WatchHandle) error {
	defer close(handle.Out)

	if _, ok := ino.wds[handle.wd]; !ok {
		return errors.New("watch not found")
	}

	ino.mu.Lock()
	delete(ino.wds, handle.wd)
	ino.mu.Unlock()

	_, err := syscall.InotifyRmWatch(ino.fd, uint32(handle.wd))
	if err != nil {
		return err
	}
	return nil
}

func inotifyReceive(ino *Inotify) {
	buf := make([]byte, InotifyBufferSize)
	for {
		buf = buf[:cap(buf)]

		n, err := syscall.Read(ino.fd, buf)
		if err != nil {
			// XXX do something with err
			return
		}
		buf = buf[:n]

		offset := 0
		for offset < len(buf) {
			event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			nameOffset := offset + syscall.SizeofInotifyEvent

			var name string
			if event.Len > 0 {
				raw := buf[nameOffset : nameOffset+int(event.Len)]
				name = strings.TrimRight(string(raw), "\x00")
			}

			// XXX contention?
			ino.mu.Lock()
			if w, ok := ino.wds[int(event.Wd)]; ok {
				w.out <- Event{
					Wd:     event.Wd,
					Cookie: event.Cookie,
					Mask:   event.Mask,
					Name:   name,
				}
			}
			ino.mu.Unlock()

			offset += syscall.SizeofInotifyEvent + int(event.Len)
		}
	}
}
