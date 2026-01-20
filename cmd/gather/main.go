package main

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mdsn/gather/lib/api"
	"github.com/mdsn/gather/lib/source"
	"github.com/mdsn/gather/lib/source/manager"
)

const SockPath = "/tmp/gather"
const SockBacklog = 1

func main() {
	log.SetFlags(0)
	log.SetPrefix("gather: ")

	// Set up a unix domain socket for ctl
	sfd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Fatalf("socket: %v", err)
	}

	// Create the Sockaddr for the UNIX domain socket
	addr := unix.SockaddrUnix{Name: SockPath}

	// Bind
	err = unix.Bind(sfd, &addr)
	if err != nil {
		log.Fatalf("bind: %v", err)
	}

	// Listen
	err = unix.Listen(sfd, SockBacklog)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	printInfo()

	// Set a handler for SIGTERM, SIGINT to cancel the root context.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	m := manager.NewManager()
	defer m.Close()

	cmdC := make(chan *api.Command)
	go read(sfd, cmdC)
	go execute(ctx, cmdC, m)
	// XXX call drain() synchronously to use it as a blocking barrier. Since
	// read() is not context-aware it does not get canceled by the signal setup
	// above, and the process never exits.
	drain(ctx, m)
}

func read(sfd int, cmdC chan *api.Command) {
	for {
		cfd, _, err := unix.Accept(sfd)
		if err != nil {
			log.Printf("accept: %v", err)
			return // XXX should signal main() to exit
		}

		// Wrap conn fd in a go *File
		cf := os.NewFile(uintptr(cfd), "")
		reader := bufio.NewReader(cf)

		// XXX make this read ctx-cancellable
		line, err := reader.ReadString('\n')

		// Reader returns an error if input did not end in \n. Ignore any
		// partial input.
		if err == io.EOF {
			cf.Close()
			continue
		}

		line = strings.TrimSpace(line)
		cmd, err := api.ParseCommand(line)
		if err != nil {
			log.Printf("parse: %v", err)
			cf.Close()
			continue
		}

		cf.Close()
		cmdC <- cmd
	}
}

func execute(ctx context.Context, cmdC chan *api.Command, m *manager.Manager) {
	for cmd := range cmdC {
		switch cmd.Kind {
		case api.CommandKindAdd:
			spec := makeSpec(cmd)
			err := m.Attach(ctx, spec)
			if err != nil {
				log.Printf("attach: %v", err)
			} else {
				log.Printf("attached source '%s'", cmd.Id)
			}
		case api.CommandKindRm:
			err := m.Remove(cmd.Id)
			if err != nil {
				log.Printf("remove: %v", err)
			} else {
				log.Printf("removed source '%s'", cmd.Id)
			}
		default:
			log.Fatalln("execute: unknown command kind")
		}
	}
}

func drain(ctx context.Context, m *manager.Manager) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-m.Events:
			fmt.Printf("%s: %s\n", ev.Id, string(ev.Bytes))
		}
	}
}

func makeSpec(cmd *api.Command) *source.Spec {
	// func NewSpec() ?
	spec := &source.Spec{
		Id:   cmd.Id,
		Path: cmd.Path,
		Args: cmd.Args,
	}

	switch cmd.Target {
	case api.CommandTargetFile:
		spec.Kind = source.KindFile
	case api.CommandTargetProc:
		spec.Kind = source.KindProc
	default:
		log.Fatalln("makeSpec: unknown command target")
	}

	return spec
}

func printInfo() {
	log.Printf("pid %d", os.Getpid())
	cwd, _ := os.Getwd()
	log.Printf("cwd %s", cwd)
}
