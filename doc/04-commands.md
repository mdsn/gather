# Commands API

The API surface is minimal. The `add` command has the following syntax:

    add (file | proc) id path [arguments...]

Arguments are ignored for a `file` target, and optional for a `proc` target.

The `rm` command uses the id:

    rm id

The application parses lines from stdin and interprets the commands. Malformed
input results in an error being reported to stderr.

## Command type

Commands are parsed into the following type

```go
type CommandKind uint8
const (
  CommandKindAdd CommandKind = iota
  CommandKindRm
)

type CommandTarget uint8
const (
	CommandTargetFile CommandTarget = iota
	CommandTargetProc
)

type Command struct {
  kind CommandKind
  target CommandTarget
  id string
  path string
  args []string
  sentAt time.Time
}
```

## Design

The application spins up a goroutine to block on stdin, parse one command at a
time, and send either the typed `Command` or an error through a channel. A
separate goroutine sits on the other end of the command/error channel. On
successfully parsed commands it manipulates the `*Manager` to attach/remove
sources, on errors it prints to stderr.
