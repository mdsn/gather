# Input semantics

Recall the example interaction from 01-design:

    # Create a fifo to give nx commands
    $ mkfifo /tmp/nxctl

    # Run nx with its input source
    $ nx < /tmp/nxctl

    # Tail syslog
    $ echo 'add file syslog /var/log/syslog' > /tmp/nxctl

The semantics of FIFOs have `echo` open the file, write to it then close it.
The reader will see EOF at that point, and would normally cause an input loop
to exit. This is not what we want in this case. The most reasonable workaround
is to ignore EOF and only exit `nx` by signal delivery.
