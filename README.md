latch
=======

https://godoc.org/github.com/glycerine/latch

Latch explores the question: what if
Go channels could be closed multiple
times, and be provided with a value
other than the zero value, as their
"closed" value.

Such channels would act like latches.

This is half of how I end up
using channels anyway. Typical
use case is for coordinating
shutdown or restart of a subsystem.

A latch holds its value, for as
many readers as want to see it, for
as long as the writer wishes. The
user can open the latch, which
means that it is empty and readers
will block like an open size-0 channel
with no current writers. The
user can also write a new value
to the latch, which will replace
the old value for all subsequent
reads.

Go channels aren't a perfect fit, but
we make them work at some cost in
space and time efficiency.

It would be great to be able to have
actual latch type in Go, so that space and time
could be optimal. This could be as simple
as taking -1 as the size of the channel to
indicate that the channel is a latch.
Such a change would be backwards
compatible, because no current correct
Go programs can use a size of -1.

The advantage of having this built
into the language would be
type safety (intead of using interface{}
in Packet), and efficiency (no
need to run a background goroutine);
both time efficiency( no need to wait
for the background goroutine to wake
up to service all your reads), and
space efficiency: no need to allocate
a bunch of sz slots in your channel,
when you only need one. There would
be less load on the Garbage Collector
as a result of not having to discard
and re-make already closed channels
and sub-systems.

----

This is free and unencumbered software released into the public domain
by author Jason E. Aten, Ph.D.

See the unlicense text in the LICENSE file.
