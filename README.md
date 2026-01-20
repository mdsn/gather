# gather

Gather follows sources of line-based output. It can tail processes or files.
It's controlled via a UNIX domain socket placed at `/tmp/gather`.

The API is simple. It has two commands: `add` and `rm`. Output is printed to
stdout, prefixed by the given source id. Application output and errors are
logged to stderr, separate from source output.

Lines are truncated at 4k bytes.

## Usage

Example:

    # Send commands to gather with socat via the socket
    echo 'add file syslog /var/log/syslog' | socat - UNIX-CONNECT:/tmp/gather
    echo 'add proc hello echo hello from a process' | socat - UNIX-CONNECT:/tmp/gather
    echo 'rm syslog' | socat - UNIX-CONNECT:/tmp/gather

## Internals

File sources are added to an inotify watch list. The files are not tracked by
name but by inode, meaning `rename(2)` and `unlink(2)` do not affect an
attached source. That also means it's not possible to track a file through
rotation. Truncation logic is best effort: if a writer does `truncate(2)`
followed by a quick write, before the first inotify event can be read, Linux
coalesces the two `IN_MODIFY` events and the fact that the file was truncated
may be lost depending on the number of bytes that result in the file. The
inotify fd is polled with `epoll(7)` along with an `eventfd(2)` that is used as
a side channel to interrupt the blocking read on the epoll fd.

Processes are spawned with Go's `os/exec` library with stdout piped back to the
parent.

## Dependencies

Inotify, epoll, eventfd. This pretty much makes `gather` Linux-only.

## Not implemented

Other sources of output are conceivable, like sockets. An `ls` command would be
nice to list all currently attached sources. File sources are not tracked
through renames, as they are followed by inode.

