# Known bugs

## `SIGINT` interrupting a system call

    ^Cpanic: EpollWait: interrupted system call

    goroutine 7 [running]:
    github.com/mdsn/gather/lib/watch.inotifyReceive(0x4000190000)
            /home/mariano/src/gather/lib/watch/inotify.go:175 +0x3f4
    github.com/mdsn/gather/lib/watch.NewInotify.func1()
            /home/mariano/src/gather/lib/watch/inotify.go:101 +0x20
    sync.(*WaitGroup).Go.func1()
            /usr/local/go/src/sync/waitgroup.go:239 +0x4c
    created by sync.(*WaitGroup).Go in goroutine 1
            /usr/local/go/src/sync/waitgroup.go:237 +0x70

## Lost output when running a subshell

    echo 'add proc subshell sh -c "echo one; echo two; echo -n three"' > /tmp/myfifo

## `SIGINT` not respected sometimes

Sometimes after output is printed it's not possible to kill `gather` with ^C.
Some goroutine is probably left hanging that prevents the process from exiting.

