# Known bugs

## Command `read()`ing polls after first command

When redirecting stdin to a FIFO, the first read from stdin blocks. When the
shell writes something to the FIFO, it opens, writes then closes it. The reader
gets the bytes, then goes back to reading but since there are no more writers
it constantly sees 0, EOF, and it turns into constant polling.

The solution is to use a AF_UNIX socket instead, block on `accept()`, read read
read, then go back to blocking on accept once the peer closes the connection.

## Lost output when running a subshell

    echo 'add proc subshell sh -c "echo one; echo two; echo -n three"' > /tmp/myfifo

## `SIGINT` not respected sometimes

Sometimes after output is printed it's not possible to kill `gather` with ^C.
Some goroutine is probably left hanging that prevents the process from exiting.

