# truncate(2)

In principle, inotify delivers `IN_MODIFY` when a file is truncated. If the file is truncated to a shorter length, and if the relevant event is read in time, it is possible to determine the new file size is less than the current offset into the file and update the offset accordingly.

Unfortunately this cannot be guaranteed. If a writer performs a write immediately after truncate, fast enough that the watch has no time to read the original `IN_MODIFY` delivered because of truncate, the event produced by the `write(2)` gets coalesced with the first one and the watch sees a single event.

So the event produced by truncate could be lost. The situation is really worse than that; suppose inotify had some toggle that prevented an instance from coalescing events. Even with that hypothetical inotify, the events have no indication of the reason for their delivery. All they carry is their watch descriptor, mask, cookie and name. They do not say "the file was truncated to 30 bytes" followed by "57 bytes were written to the file". If both events were available without coalescing, the fact that file was truncated before getting more bytes would be lost.

Thus, without a more sophisticated mechanism like eBPF producing a more detailed log, no guarantee can be made that `truncate(2)` and subsequent `write(2)` calls will be processed in the "natural" way. It is possible for abnormal output to be produced in racy scenarios. For example, suppose there is a file with the following 57 bytes of content:

    To those who gaze on thee what language could they speak?

Then attach the watch, truncate it to 0, and write 64 bytes:

    You can lead a horse to water, but you can't make him backstroke

Suppose both events are coalesced, then the watch reads an `IN_MODIFY`. It sees its offset is 57 but the new file size is 64, assumes 7 bytes were written to the file and outputs:

    kstroke

This illustrates the kind of things that can happen because of the loss of information of the previous states of the file.
