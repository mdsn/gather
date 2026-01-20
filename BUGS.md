# Known bugs

## Lost output when running a subshell

    echo 'add proc subshell sh -c "echo one; echo two; echo -n three"' > /tmp/myfifo

## `SIGINT` not respected sometimes

Sometimes after output is printed it's not possible to kill `gather` with ^C.
Some goroutine is probably left hanging that prevents the process from exiting.

## Proc sources are not cleaned up once the process exits

They remain in the state. This is unimplemented logic rather than a bug.
