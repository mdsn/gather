package proc

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/mdsn/nexus/lib/source"
)

var (
	streamGracePeriod = time.Second
)

func Attach(ctx context.Context, spec *source.Spec) (*source.Source, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)

	// Create pipes
	rp, wp, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// TODO 01-design says both stdin and stderr are streamed and aggregated.
	// Assign write end to the child
	cmd.Stdout = wp

	// Fork/exec
	if err := cmd.Start(); err != nil {
		rp.Close()
		wp.Close()
		return nil, err
	}

	// Close parent copy of the write pipe
	if err := wp.Close(); err != nil {
		rp.Close()
		return nil, err
	}

	// Create *Source instance
	src := &source.Source{
		Id:   spec.Id,
		Kind: source.KindProc,
		// XXX Ready barrier?
		Done:   make(chan struct{}),
		Out:    make(chan source.Output),
		Err:    make(chan error),
		Cancel: cancel,
	}

	// Start streaming output into the channel
	st := stream(rp, src)

	// Wait out the process in a goroutine
	go func(grace time.Duration) {
		defer close(src.Done)

		// TODO report this error?
		_ = cmd.Wait()

		// Drain pipe with 1 second of grace, then shut it down and wait for
		// streaming Done signal.
		select {
		case <-ctx.Done():
			close(st.Stop)
		default:
			timer := time.NewTimer(grace)
			defer timer.Stop()
			select {
			case <-st.Done:
				// Nothing
			case <-timer.C:
				close(st.Stop)
			}
		}

		<-st.Done
		// TODO possibly build a "exit status" for the source
	}(streamGracePeriod)

	return src, nil
}

type ProcStream struct {
	Done chan struct{}
	Stop chan struct{}
}

func stream(pipe io.ReadCloser, src *source.Source) *ProcStream {
	st := &ProcStream{
		Done: make(chan struct{}),
		Stop: make(chan struct{}),
	}

	go read(pipe, src, st)
	go cleanup(pipe, src.Out, st)

	return st
}

func read(pipe io.Reader, src *source.Source, ctl *ProcStream) {
	// Signal that streaming is done.
	defer close(ctl.Done)

	rd := bufio.NewReaderSize(pipe, source.MaxLineLength)
	for {
		// This call blocks until the pipe is closed. Includes delimiter.
		bytes, err := rd.ReadSlice('\n')

		n := len(bytes)
		if n > 0 {
			if bytes[n-1] == '\n' {
				// ReadSlice includes the newline
				n--
			}

			cp := make([]byte, n)
			copy(cp, bytes[:n])

			msg := source.Output{Id: src.Id, CapturedAt: time.Now(), Bytes: cp}

			// Preempt writing if a Stop signal arrived.
			select {
			case src.Out <- msg:
			case <-ctl.Stop:
				return
			}
		}

		// The buffer filled before reaching newline. Discard the rest of the
		// line.
		for errors.Is(err, bufio.ErrBufferFull) {
			_, err = rd.ReadSlice('\n')
		}

		if errors.Is(err, io.EOF) {
			return
		}

		if err != nil {
			return // XXX do something
		}
	}
}

// Close the pipe and out channel.
func cleanup(pipe io.Closer, out chan source.Output, st *ProcStream) {
	defer close(out)
	select {
	// Streaming goroutine exited on its own. Close the pipe and get out.
	case <-st.Done:
		_ = pipe.Close()
	// Stop signal arrived. Close the pipe to unblock a ReadBytes(), then wait
	// for the streaming goroutine to be done.
	case <-st.Stop:
		_ = pipe.Close()
		<-st.Done
	}
}
