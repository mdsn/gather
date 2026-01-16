package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mdsn/nexus/lib/api"
	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/source/manager"
)

func main() {
	// Set a handler for SIGTERM, SIGINT to cancel the root context.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	m := manager.NewManager()
	defer m.Close()

	cmdC := make(chan *api.Command)
	go read(cmdC) // ctx?
	go execute(ctx, cmdC, m)

	fmt.Println("lol")
}

func read(cmdC chan *api.Command) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			// Ignore EOF and any partial line; stdin may be redirected to the
			// read end of a FIFO, which may produce multiple EOF as writers
			// open and close it. See 05-input-semantics.
			continue
		}

		cmd, err := api.ParseCommand(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse: %v", err)
			continue
		}

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
				fmt.Fprintf(os.Stderr, "execute: attach: %v", err)
			}
			// TODO log attach to stderr?
		case api.CommandKindRm:
			// ...
		default:
			panic("execute: unknown command kind")
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
		panic("makeSpec: unknown command target")
	}

	return spec
}
