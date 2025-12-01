# Design doc: nexus

Nexus fans-in multiple line-based sources into one log. The sources may be
files or processes. Sources may be added or removed dynamically without
interrupting a tailing session.

## Design approach

The project is implemented as a single Go binary. It receives commands from a
Unix domain socket it creates on startup. Nexus understands a simple text based
protocol via this socket. The API is intentionally small; it covers adding and
removing each kind of source.

A library implements the functionality to attach to sources and forward their
output to the main binary.

  cmd/nx                The main binary
  lib/source            Library to get output from file and process sources
  lib/api               Command API via UNIX domain socket

### cmd/nx

The binary acts as a coordinator. It receives commands, acts on them to set up
and tear down sources, and prints output from attached sources. It listens to
SIGTERM to wrap things up when the binary is terminated.

This is where the program state lives. It is a map of source id to a `Source`
type. The binary takes care of validating the source id to ensure uniqueness.

```go
  type State struct {
    Sources map[string]*source.Source
  }
```

### lib/source

The state of the program consists of a map of the sources currently attached
and being tailed. When a command to attach a new source arrives, the path or
command is passed over to the `source` library to set up the necessary
machinery and obtain a channel through which output is received.

The library takes care of closing the channel when the source context is done.
For proc sources, it terminates child processes when the context is done. For
file sources, it closes the file when the context is done.

```go
  type SourceKind uint8

  const (
      SourceFile SourceKind = iota
      SourceProc
  )

  type Output struct {
    Id          string
    CapturedAt  time.Time
    Bytes       []byte
  }

  type Source struct {
    Id    string
    Kind  SourceKind
    Out   chan Output
    Err   chan error
  }

  // The main binary constructs a Spec instance from a Command of kind Add, as
  // input to the `Attach` function.
  type Spec struct {
    Id    string
    Kind  SourceKind
    Path  string
    Args  []string
  }

  // Start tailing the requested source and return a *Source to represent it.
  // The provided context controls the lifetime of the file watch or process
  // run. Attach dispatches on the Kind to set up either a follow on a file or
  // a process.
  func Attach(ctx context.Context, spec *Spec) (*Source, error)
```

### lib/api

The API library provides functions to create the socket and parse commands into
types. The main binary does not operate in terms of strings; it understands the
types provided by `lib/api`.

```go
  type CommandTarget uint8
  const (
      TargetFile CommandTarget = iota
      TargetProc
  )

  type CommandKind uint8
  const (
    KindAdd CommandKind = iota
    KindRm CommandKind
  )

  type Command {
    Kind    CommandKind
    Target  CommandTarget
    Id      string          // Unique identifier for this source
    Input   string          // File path or command name
    Args    []string        // Arguments for Proc target
  }
```

A command handler will dispatch based on `Kind` to concrete functions to handle
`Add` or `Rm` commands.

## UX

These are command examples:

    # Tail syslog
    $ echo 'add syslog file /var/log/syslog' | nc -U /tmp/nexus.sock

    # Tail a hypothetical worker
    $ echo 'add worker run -- ./worker --queue=emails' | nc -U ...

    # Remove both
    $ echo 'rm syslog' | nc -U ...
    $ echo 'rm worker' | nc -U ...

The command is of the form `add <id> <type> <path>`.

### Output

Output is line based, headed by the id given to the source. Output is assumed
to be UTF-8; binary or otherwise invalid UTF-8 will be passed through raw.
Output is split on `\n`; output over some yet undetermined length is truncated.

    syslog: one log line in syslog.
    worker: working, working, all good.

## API

Nexus understands a simple API via a Unix domain socket. These commands
constitute the API:

    add myFile file /path/to/file       Start tailing a new file.
    rm myFile                           Stop tailing a file.
    add myCmd run -- cmd arg1 arg2      Run a command and tail its output.
    rm myCmd                            Terminate a command to stop tailing it.

The `add` command takes a type, which can be `file` or `proc`, a string
identifier and the concrete source. For files, the source is the path; for
processes it's the command and arguments to run as a child process.

The identifier is unique among all sources. It identifies the tailed subject in
the aggregated log and in `rm` commands.

## Scope

There are genuine sources of lines other than files and processes, for example
named pipes and sockets. Nexus only accepts files and processes.

An interactive UI would allow the user to enter commands directly, in the style
of less or vi. Instead of implementing a full fledged line editor, nexus gets
input via a Unix socket.

For process sources, nexus captures stdout and stderr. It logs everything to
stdout. When terminating a running process, only the spawned child is
terminated. The only fireproof way to terminate an entire process tree is to
run them in a pid namespace, which is out of the scope of this design.
Therefore, if a process spawns its own children, those are not terminated when
stopping the root process.

For file sources, file rotation is not supported. The file descriptor is kept
open through moves, but events in the directory or files are not watched, so
the behavior of `tail -F` is not implemented.

## Implementation details

### Tailing subjects identifiers

The user provides a unique identifier for each source they start tailing. The
`rm` command then identifies the source to remove with that identifier.

### Processes stdout/stderr descriptors

Processes that write to stdout and stderr have their output aggregated.
Conceivably, each file descriptor could be identified in the log, or the output
of nexus itself could be written to stdout/stderr correspondingly, but this is
the simplest option.

### Process output

Binary output is not forbidden, but it makes little sense to aggregate binary
output and print it headed with the textual id of each source.

### Tool output

Nexus reserves stdout for the aggregated output of its sources. It prints its
own output messages to stderr.
