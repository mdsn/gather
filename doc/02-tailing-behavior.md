# File tailing behavior

File watches are based on `inotify(2)`. The file Source keeps a file descriptor open and an offset into the file; when an event arrives, the file size is updated with `fstat(2)` to know if the file was truncated or there is new data to read.

Keeping the fd open interferes with the way the OS treats the inode. If `unlink(2)` is called on a file, the OS does not delete it because of the open file descriptor. Inotify does not deliver a `IN_DELETED_SELF` event at `unlink(2)` time if the inode cannot be destroyed yet.

An alternative arrangement would have the Source maintain an offset and path, and `open(2)`, `read(2)` or `pread(2)`, and `close(2)` the file every time an event arrives. This increases the odds for races between calls, and adds complexity for a scenario where a file is rotated before the relevant events are processed. Inotify events may also be lost due to queue overrun, which makes things more complicated.

Another aspect of this situation is that even if the watch were able to detect that a file was deleted, there is no way to tell whether other processes have fds open to that file. The file might continue to grow indefinitely, even though it has become anonymous, so long as file descriptors to it are kept open. Thus, there is no certain way to tell that a file is "ready to be removed" even when we can see its link count go to 0; the OS does not give the number of open fds to an inode.

This is probably the reason why GNU `tail(1)` makes no promises around rename/unlink when following a descriptor:

    When following a descriptor, tail does not detect that the file has been unlinked or renamed and issues no message; even though the file may no longer be accessible via its original name, it may still be growing.

It does detect that a file has been removed when called with `-f name`:

    When following by name, tail can detect that a file has been removed and gives a message to that effect [...]

The natural way to detect this is to store the file's inode when first opening it, then checking it every time there is an event and the file is reopened. If the inode changed, the file has been rotated (either renamed or unlinked).

The design doc explicitly says file rotation is not supported. So for now we will keep it as is, with the file descriptor open throughout the watch, and intentionally not attempt to detect file deletion to stop the watch when it happens.
