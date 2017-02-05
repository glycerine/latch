the latch proposal (latch would be a new type in Go)
----------------------------------------------------

2017 February 5

A latch is like a channel that broadcasts a user-defined
value. Broadcasts can be done by closing channels in Go, but they
can only convey a single bit: the zero-value or blocked (still open).
Moreover the channel cannot be efficiently re-used, and
we must create load on the garbage collector by re-making
channels if we want to broadcast again; and even then
correctness is hard because we must shutdown all sub-
systems that may be selecting on the old instance of that channel.

What if we had a channel:

* that could convey different values once they are closed.

* that could be closed more than once.

* that could be re-opened more than once.

* that didn't need a background goroutine to service it.

I call this a latch.

The library code here provides a working latch prototype. It is
not as efficient or as type safe as built-in latch.

~~~
   // creation syntax is backwards compatible with Go 1
   //
   // Only change: now -1 can be passed as the channel size.
   //
   latch := make(chan int, -1)

   <-latch    // this read would block. the latch is open by default.

   close(latch) // now the latch always returns 0, just like a channel.

   // writes are allowed after close.
   // close can be called multiple times.
   // writes to a latch never block.
   // reads to a latch block if the latch is open.
   //
   latch <- 3 // now latch always returns 3.

   // each of these returns 3 immediately (no blocking)
   three := <- latch
   tres  := <- latch
   trois := <- latch

   latch <- 2 // now all readers see 2

   // each of these returns 2 immediately (no blocking)
   two  := <- latch
   dos  := <- latch
   deux := <- latch

   // without an open() operation, how do we open a latch?
   latch <- 0 // writing the zero-value would open the latch.

   select {
     case <-latch:    // this now blocks, because the latch is "open".
     default:
        fmt.Printf("opened latches block") // this would print.
   }

   // this would "close" the latch again; all readers would see 1.
   latch <- 1

   close(latch) // now the latch returns the zero-value all the time, immediately: this is just like a regular channel.

   // since there is no open keyword, we
   // define writing the zero value to mean open
   // the latch, and define close(latch) to
   // mean write the zero-value to the latch.
   // A little strange, but preserves backwards
   // compatiblity with Go 1.
   //
   latch <- 0  // now the latch would block readers, because it is in "open" state.

~~~

Note: It would be preferred to have an `open(latch)` syntax, but
that is deferred because it requires a new language keyword.

background
----------

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
