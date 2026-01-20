# Input semantics II

Reading commands from stdin and having the shell redirect a FIFO to it is
simple and easy to understand. Unfortunately the semantics of reading make for
less than ideal behavior. To wit, the first `read(2)` on stdin blocks, but
after a command arrives, the shell closes the FIFO and gather reads EOF; there
are no writers on the other end, so all that gather can do is attempt to
`read(2)` again, except that the subsequent calls do not block. Instead they
return immediately with 0 bytes and EOF, which means constant useless polling
on stdin. Keeping a sentinel writer or closing and reopening stdin removes any
simplicity earned by the mechanism in the first place.

Instead, it is easier to use a UNIX domain socket. Gather now creates one at
`/tmp/gather`. `accept(2)` blocks on it, then reads are correctly separated
between commands.

The example interaction from 01-design and 05-input-semantics is easily updated:

    $ gather

    # Write via socat
    $ echo 'add file syslog /var/log/syslog' | socat - UNIX-CONNECT:/tmp/gather

