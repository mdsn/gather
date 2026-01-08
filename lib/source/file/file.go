package file

import (
	"context"
	"io"
	"os"

	"github.com/mdsn/nexus/lib/lines"
	"github.com/mdsn/nexus/lib/source"
	"github.com/mdsn/nexus/lib/watch"
)

func fileSize(fp *os.File) (int64, error) {
	stat, err := fp.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

func Tail(ctx context.Context, src *source.Source, fp *os.File, evC chan watch.Event) {
	defer close(src.Done)

	// Start at EOF
	offset, err := fileSize(fp)
	if err != nil {
		return // XXX ?
	}

	lb := lines.NewLineBuffer(4096 * 2)
	buf := make([]byte, 4096)

	// Start listening
	close(src.Ready)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case <-evC:
			// Continue
		}

		sz, err := fileSize(fp)
		if err != nil {
			return
		}

		// File was truncated, bring offset back
		if sz < offset {
			offset = sz
			continue
		}

		// read file from offset
		_, err = fp.Seek(offset, 0)
		if err != nil {
			return
		}

		for {
			buf = buf[:cap(buf)]

			n, err := fp.Read(buf)
			if n == 0 && err == io.EOF {
				break
			}
			if err != nil {
				return
			}

			offset += int64(n)

			lb.Add(buf)
			for line := range lb.Lines() {
				src.Send(line)
			}
		}
	}
}
