package api

import (
	"testing"
)

func TestParseCommand_ErrorScenarios(t *testing.T) {
	tests := []string{
		"",
		" ",
		"append file /var/log/syslog",
		"add proc",
		"add proc myWorker",
		"add file",
		"add file myFile",
		"add rhubarb mySalad",
		"rm",
	}

	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			cmd, err := ParseCommand(tc)
			if cmd != nil {
				t.Fatalf("expected nil command, got: %v", cmd)
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestParseCommand_AddFile(t *testing.T) {
	cmd, err := ParseCommand("add file myFile /var/log/syslog")
	if err != nil {
		t.Fatalf("got err: %v", err)
	}

	if cmd == nil {
		t.Fatal("nil cmd")
	}

	if cmd.Kind != CommandKindAdd {
		t.Fatal("wrong command kind")
	}

	if cmd.Target != CommandTargetFile {
		t.Fatal("wrong command target")
	}

	if cmd.Id != "myFile" {
		t.Fatal("wrong command id:", cmd.Id)
	}

	if cmd.Path != "/var/log/syslog" {
		t.Fatal("wrong command path:", cmd.Path)
	}

	if cmd.sentAt.IsZero() {
		t.Fatal("sentAt not initialized")
	}

	if len(cmd.Args) != 0 {
		t.Fatal("unexpected args:", cmd.Args)
	}
}

func TestParseCommand_AddProc(t *testing.T) {
	tests := []struct {
		name string
		in   string
		argc int
	}{
		{"with-args", "add proc myWorker ./worker -v /etc/worker.conf", 2},
		{"no-args", "add proc myWorker ./worker", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := ParseCommand(tc.in)
			if err != nil {
				t.Fatalf("got err: %v", err)
			}

			if cmd == nil {
				t.Fatal("nil cmd")
			}

			if cmd.Kind != CommandKindAdd {
				t.Fatal("wrong command kind")
			}

			if cmd.Target != CommandTargetProc {
				t.Fatal("wrong command target")
			}

			if cmd.Id != "myWorker" {
				t.Fatal("wrong command id:", cmd.Id)
			}

			if cmd.Path != "./worker" {
				t.Fatal("wrong command path:", cmd.Path)
			}

			if cmd.sentAt.IsZero() {
				t.Fatal("sentAt not initialized")
			}

			if len(cmd.Args) != tc.argc {
				t.Fatal("wrong number of args:", cmd.Args)
			}
		})
	}
}

func TestParseCommand_Rm(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"with-args", "rm myWorker ignored-arg onetwothree"},
		{"no-args", "rm myWorker"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
		})
		cmd, err := ParseCommand(tc.in)
		if err != nil {
			t.Fatalf("got err: %v", err)
		}

		if cmd == nil {
			t.Fatal("nil cmd")
		}

		if cmd.Kind != CommandKindRm {
			t.Fatal("wrong command kind")
		}

		if cmd.Target != CommandTargetUnknown {
			t.Fatal("wrong command target")
		}

		if cmd.Id != "myWorker" {
			t.Fatal("wrong command id:", cmd.Id)
		}

		if len(cmd.Path) != 0 {
			t.Fatal("unexpected command path")
		}

		if cmd.sentAt.IsZero() {
			t.Fatal("sentAt not initialized")
		}

		if len(cmd.Args) != 0 {
			t.Fatal("unexpected args:", cmd.Args)
		}
	}

}
