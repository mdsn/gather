package source

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

func attachProc(ctx context.Context, spec *Spec) (*Source, error) {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)

	// Create pipes
	rp, wp, err := os.Pipe()
	if err != nil {
		return nil, err
	}

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
	src := &Source{
		Id: spec.Id,
		Kind: KindProc,
		Done: make(chan struct{}),
		Out:  make(chan Output),
		Err:  make(chan error),
	}

	// Start streaming output into the channel
	st := stream(rp, src.Out)

	// Wait out the process in a goroutine
	go func() {
		defer close(src.Done)

		// TODO report this error?
		_ = cmd.Wait()

		// Drain pipe with 1 second of grace, then shut it down and wait for
		// streaming Done signal.
		select {
		case <-ctx.Done():
			close(st.Stop)
		default:
			timer := time.NewTimer(time.Second)
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
	}()

	return src, nil
}

type ProcStream struct {
	Done chan struct{}
	Stop chan struct{}
}

func stream(pipe io.ReadCloser, out chan Output) *ProcStream {
	st := &ProcStream{
		Done: make(chan struct{}),
		Stop: make(chan struct{}),
	}

	go read(pipe, out, st.Stop, st.Done)
	go cleanup(pipe, out, st)

	return st
}

func read(pipe io.Reader, out chan<- Output, stop <-chan struct{}, done chan<- struct{}) {
	// Signal that streaming is done.
	defer close(done)

	rd := bufio.NewReader(pipe)
	for {
		// This call blocks until the pipe is closed.
		bytes, err := rd.ReadBytes('\n')

		n := len(bytes)
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, bytes[:n])

			msg := Output{CapturedAt: time.Now(), Bytes: cp}

			// Preempt writing if a Stop signal arrived.
			select {
			case out <- msg:
			case <-stop:
				return
			}
		}

		if err == io.EOF {
			return
		}

		if err != nil {
			return // XXX do something
		}
	}
}

// Close the pipe and out channel.
func cleanup(pipe io.Closer, out chan Output, st *ProcStream) {
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
