package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/mdsn/nexus/lib/api"
)

func main() {
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
