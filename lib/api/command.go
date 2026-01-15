package api

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type CommandKind uint8

const (
	CommandKindAdd CommandKind = iota
	CommandKindRm
)

type CommandTarget uint8

const (
	CommandTargetUnknown = iota
	CommandTargetFile
	CommandTargetProc
)

type Command struct {
	kind   CommandKind
	target CommandTarget
	id     string
	path   string
	args   []string
	sentAt time.Time
}

func ParseCommand(in string) (*Command, error) {
	toks := tokens(in)
	if len(toks) == 0 {
		return nil, errors.New("empty input")
	}

	switch toks[0] {
	case "add":
		return parseAdd(toks[1:])

	case "rm":
		return parseRm(toks[1:])

	default:
		return nil, errors.New(fmt.Sprintf("unknown command '%s", toks[0]))
	}

	return nil, nil
}

func parseAdd(toks []string) (*Command, error) {
	if len(toks) < 3 {
		return nil, errors.New("missing arguments to 'add'")
	}

	cmd := &Command{
		kind:   CommandKindAdd,
		id:     toks[1],
		path:   toks[2],
		sentAt: time.Now(),
	}

	switch toks[0] {
	case "file":
		cmd.target = CommandTargetFile
		return cmd, nil
	case "proc":
		cmd.target = CommandTargetProc
		cmd.args = toks[3:]
		return cmd, nil
	default:
		return nil, errors.New(fmt.Sprintf("add: unknown source type '%s'", toks[0]))
	}
}

func parseRm(toks []string) (*Command, error) {
	if len(toks) < 1 {
		return nil, errors.New("missing argument to 'rm'")
	}

	cmd := &Command{
		kind:   CommandKindRm,
		id:     toks[0],
		sentAt: time.Now(),
	}

	return cmd, nil
}

func tokens(in string) []string {
	parts := strings.Split(in, " ")
	return slices.DeleteFunc(parts, func(s string) bool {
		return len(s) == 0
	})
}
