package watch

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	InotifyBufferSize = 4096
	// TODO IN_IGNORED can also happen when the os clears a watch due to delete
	// or unmount. Remove watch when that happens.
	InotifyMask = unix.IN_MODIFY
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
	// synchronizes access to wds
	mu sync.Mutex
	// awaits the inotifyReceive goroutine
	wg sync.WaitGroup
	// ensure a single call to Close()
	once sync.Once
	// inotify descriptor
	ifd int
	// eventfd(2) descriptor
	evfd int
	// epoll descriptor
	epfd int
	// watches keyed by watch descriptor
	wds map[int]*Watch
	// Indicates inotifyReceive goroutine exited
	done chan struct{}
}

func NewInotify() (*Inotify, error) {
	ifd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}

	evfd, err := unix.Eventfd(0, unix.EFD_CLOEXEC|unix.EFD_NONBLOCK)
	if err != nil {
		return nil, err
	}

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, ifd, &unix.EpollEvent{
		Events: unix.EPOLLIN, Fd: int32(ifd),
	})
	if err != nil {
		return nil, err
	}

	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, evfd, &unix.EpollEvent{
		Events: unix.EPOLLIN, Fd: int32(evfd),
	})
	if err != nil {
		return nil, err
	}

	ino := &Inotify{
		wg:   sync.WaitGroup{},
		once: sync.Once{},
		ifd:  ifd,
		evfd: evfd,
		epfd: epfd,
		wds:  make(map[int]*Watch),
		done: make(chan struct{}),
	}
	ino.wg.Go(func() { inotifyReceive(ino) })

	return ino, nil
}

func (ino *Inotify) Close() error {
	var err error
	ino.once.Do(func() {
		defer close(ino.done)

		// Write 1 into evfd, wait for ino.done (goroutine exited) then clean up
		// all three file descriptors.
		_, err = unix.Write(ino.evfd, []byte{0, 0, 0, 0, 0, 0, 0, 1})
		if err != nil {
			panic(fmt.Sprintf("eventfd write: %v", err))
		}

		ino.wg.Wait() // Wait for inotifyReceive to wrap up.

		err = errors.Join(
			unix.Close(ino.ifd),
			unix.Close(ino.evfd),
			unix.Close(ino.epfd))
	})
	return err
}

func (ino *Inotify) Add(path string) (*WatchHandle, error) {
	wd, err := unix.InotifyAddWatch(ino.ifd, path, InotifyMask)
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

	// XXX Does `success` have any use here?
	_, err := unix.InotifyRmWatch(ino.ifd, uint32(handle.wd))
	if err != nil {
		return err
	}
	return nil
}

func inotifyReceive(ino *Inotify) {
	epev := make([]unix.EpollEvent, 2) // Max 2 fd: inotify, eventfd
	buf := make([]byte, InotifyBufferSize)
	for {
		buf = buf[:cap(buf)]

		// XXX get rid of panics, fix error handling. This runs in a WaitGroup

		// Block for either the inotify or eventfd to be ready.
		n, err := unix.EpollWait(ino.epfd, epev, -1)
		if err != nil {
			panic(fmt.Sprintf("EpollWait: %v", err))
		}

		if n <= 0 {
			panic("EpollWait: n <= 0") // Should not happen with timeout == -1
		}

		// epev contains at most two events; if one of them is on evfd, we are
		// wrapping up and need to terminate the goroutine.
		for i := range n {
			if int(epev[i].Fd) == ino.evfd {
				return
			}
		}

		// ino fd is ready and we can proceed to collect the events.
		n, err = unix.Read(ino.ifd, buf)
		if err != nil {
			panic(fmt.Sprintf("Read: %v", err))
		}
		buf = buf[:n]

		offset := 0
		for offset < len(buf) {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			nameOffset := offset + unix.SizeofInotifyEvent

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

			offset += unix.SizeofInotifyEvent + int(event.Len)
		}
	}
}
