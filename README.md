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

* that could broadcast different values;

* that was space-efficient;

* that didn't need a background goroutine to service it for correctness; to relieve the need to know how many subscribers a latch had or will have in the future;

* that could be emptied to suspend broadcast.

We call this a latch.

The library code here provides a working latch prototype. It is
not as efficient or as type safe as a built-in latch would be.

~~~
   // make a new (open) latch:
   //
   // define runtime.CreateLatch(bc) that takes a buffered
   // channel bc and returns a latch that uses the bc
   // as its data source.
   //
   backit := make(chan int, 1)
   latch := runtime.CreateLatch(backit)

   // both backit and latch can still be accessed.
   // The latch uses backit as its backing data.

   // The only difference is: receives on latch don't actually
   // remove anything from backit.

   // Receives on latch
   // always return the data that would be received from backit.

   // Receives on latch block if backit is empty, just as a
   // receive on backit would.

   // Writes to the latch have the effect of emptying
   // backit followed by a single write (send) to backit.

   latch <- 4 // equivalent to: for len(backit) > 0 { <-backit}; backit <- 4

   // we can read as much as we like:
   fmt.Printf("%v\n", <-latch) // printf "4"
   fmt.Printf("%v\n", <-latch) // printf "4"
   fmt.Printf("%v\n", <-latch) // printf "4"

   // emptying backit will empty latch
   f := <-backit 
   assert(f == 4)

   <-latch // blocks, now empty buffer behind the channel.

   latch <- 3
   
   // each of these returns 3 immediately (no blocking)
   three := <- latch
   tres  := <- latch
   trois := <- latch

   latch <- 2 // now all readers see 2

   // each of these returns 2 immediately (no blocking)
   two  := <- latch
   dos  := <- latch
   deux := <- latch

   // forget any stored value, by reading the one value out of the backing backit.
   <-backit

   select {
     case <-latch:    // this now blocks, because the backing queue is empty
     default:
        fmt.Printf("empty latches block") // this would print.
   }

   
~~~



background
----------

https://godoc.org/github.com/glycerine/latch

Latch explores the question: what if
Go channels could be used to broadcast
more than a single bit?

Image a channel that could send to multiple readers
a single current value, without having
to know how many readers there are. Imagine
being able to change that value without
destroying and re-creating all receiving structures.

Such channels would act like latches.

This is half of how I end up
using channels anyway. Typical
use case is for coordinating
shutdown or restart of a subsystem.
Go's channels are just somewhat awkward
for this purpose.

A latch holds its value, for as
many readers as want to see it.
The owner can empty the latch, which
means that readers will block.
An empty latch is like an empty
channel. Both block receivers.

The user can also write a new value
to the latch, which will replace
the old value for all subsequent
reads.

Go channels aren't a perfect fit, but
we make them work at some cost in
space and time efficiency.

It would be great to be able to have
actual latch type in Go, so that space and time
could be optimal.

The advantage of having this built
into the language would be
type safety (intead of using interface{}
in Packet), and efficiency (no
need to run a background goroutine).

Also, time efficiency would be improved
because there would be no need to wait
for the background goroutine to wake
up to service all your reads. 

Space efficiency would be improved
by not having a need to allocate
a bunch of slots in the non-blocking sized channel,
as we do in the prototype library.

There would be less load on the Go
garbage collector
as a result of not having to discard
and re-make already closed channels
and the structs/sub-systems that use
those channels receiving broadcasts.

----

This is free and unencumbered software released into the public domain
by author Jason E. Aten, Ph.D.

See the unlicense text in the LICENSE file.
