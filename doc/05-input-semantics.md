# Input semantics

Recall the example interaction from 01-design:

    # Create a fifo to give gather commands
    $ mkfifo /tmp/gatherctl

    # Run gather with its input source
    $ gather < /tmp/gatherctl

    # Tail syslog
    $ echo 'add file syslog /var/log/syslog' > /tmp/gatherctl

The semantics of FIFOs have `echo` open the file, write to it then close it.
The reader will see EOF at that point, and would normally cause an input loop
to exit. This is not what we want in this case. The most reasonable workaround
is to ignore EOF and only exit `gather` by signal delivery.
