the latched channel proposal
----------------------------------------------------

2017 February 10

Based on a new idea: having the receiver do the
the maintenance on the channel, I believe I have
solved this problem in existing Go. See the file reload.go
for an example implementation.


2017 February 5

A latched channel is a channel that supports
broadcast of a user-defined value. Broadcasts
can be done by closing regular channels in Go, but they
can only convey a single bit: the zero-value or blocked (still open).

Moreover after broadcast the regular channel cannot be efficiently re-used, and
we must create load on the garbage collector by re-making
channels if we want to broadcast again; and even then
correctness is hard because we must shutdown all sub-
systems that may be selecting on the old instance of that channel.

What if we had a channel:

* that could broadcast different values;

* that was space-efficient;

* that didn't need a background goroutine to service it for correctness; to relieve the need to know how many subscribers a latch had or will have in the future;

* that could be emptied to suspend broadcast.

We call this a latched channel. Or latch for short.

The library code here provides a latch prototype to
help the reader's comprehension. It is
not as efficient or as type safe as a built-in latch would be.
It also risks correctness and efficiency when the number
of receivers is not known and fixed in advance.
See the code at the bottom for that prototype, which only
approximates the desired semantics.

The following example shows how the proposed additional built-in
latched channel type would work. A latch would be a channel but
with an extra bit set internally that makes reads non-consuming
and tells the latch to share its backing store with a regular
buffered channel.

~~~
   // Proposed usage (ideal; not possible today):
   //
   // make a new latch:
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

   // equivalent to the atomic
   // execution of: for len(backit) > 0 { <-backit}; backit <- 4
   latch <- 4 

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
for this purpose. I'm constantly
tip-toe-ing around the lack of
support for close()-ing more than
once. I was inspired to write a library just to
make multiple-closes not crash. It improves
usability some, but latches would
be even better.
(See https://github.com/glycerine/idem
for that library).

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

-----
code

~~~
package latch

import (
	"sync"
	"time"
)

// A Latch is a channel that broadcasts a *Packet value,
// without knowing who will receive it.
// All readers will read the Bcast() value.
type Latch struct {
	sz    int
	mut   sync.Mutex
	cur   *Packet
	ch    chan *Packet
	avail bool // when avail==true, <- receives on Ch() will be given cur.

	fillerStop chan struct{}
}

// Packet conveys either a data Item,
// or an Err (or, possibly, both).
type Packet struct {
	Item interface{}
	Err  error
}

// NewLatch makes a new latch with
// backing channel of size sz.
func NewLatch(sz int) *Latch {
	return &Latch{
		ch: make(chan *Packet, sz),
		sz: sz,
	}
}

// Ch returns a read-only channel. This is
// on purpose -- we want to prevent
// anyone from putting values
// into the channel by means other than
// calling Bcast().
func (r *Latch) Ch() <-chan *Packet {
	return r.ch
}

// clients should call Clear(), not drain() directly.
// Internal callers should be holding the r.mut already.
func (r *Latch) drain() {
	if len(r.ch) == 0 {
		return
	}
	// safe for concurrent reads; in
	// case a client happens to be reading
	// now, we don't want to block inside drain().
	for {
		select {
		case <-r.ch:
		default:
			return
		}
	}
}

// Bcast can be called multiple times, with
// different values of pak. Each call will
// drain the ch channel of any prior data,
// any replace it will sz copies of pak.
// The sz value was set during NewLatch(sz).
func (r *Latch) Bcast(pak *Packet) {
	r.mut.Lock()
	r.cur = pak
	r.drain() // drop any old values.
	r.avail = true
	for i := 0; i < r.sz; i++ {
		r.ch <- r.cur
	}
	r.mut.Unlock()
}

// Refresh "tops-up" a available channel. Since
// the channel is of finite size, and
// we don't want to waste a background
// goroutine (for speed and space), clients
// can regularly call Refresh to make sure
// an in-use (Bcast called) channel still has copies
// of data. Otherwise, after sz accesses,
// receivers on Ch() will block.
//
// If you want absolute correctness, and
// can't be bothered with invoking Refresh()
// regularly to service your available channel,
// call BackgroundRefresher() once instead.
//
func (r *Latch) Refresh() {
	r.mut.Lock()
	if r.avail {
		for len(r.ch) < r.sz {
			r.ch <- r.cur
		}
	}
	r.mut.Unlock()
}

// BackgroundRefresher starts a goroutine
// that tops-up your available channel
// every 500msec. It will prevent
// starvation if you have lots of consumers;
// at the cost of using a goroutine and
// possibly slowing down your whole program.
//
// Efficiency minded clients should arrange
// to themselves invoke Refresh() regularly
// instead if at all possible; (and of course,
// only if required). We expect the need for
// BackgroundRefresher to be rare. Refresh
// and BackgroundRefresher are needed only
// is only if the number reads will
// exceed sz or cannot be known in advance.
// When using Latch as a termination
// signal, for instance, sz is typically
// bounded by the number of clients of
// those goroutines that are being shutdown.
//
func (r *Latch) BackgroundRefresher() {
	r.mut.Lock()
	defer r.mut.Unlock()
	if r.fillerStop == nil {
		r.fillerStop = make(chan struct{})
		go func() {
			for {
				select {
				case <-r.fillerStop:
					return
				case <-time.After(500 * time.Millisecond):
					r.Refresh()
				}
			}
		}()
	}
}

// Stop tells any BackgroundRefresher goroutine
// to shut down.
func (r *Latch) Stop() {
	r.mut.Lock()
	defer r.mut.Unlock()
	if r.fillerStop != nil {
		// only close it once.
		select {
		case <-r.fillerStop:
		default:
			close(r.fillerStop)
		}
	}
}

// Clear drains the latch, emptying
// it of any values stored. After
// we return, receivers on Ch()
// will block until somebody
// calls Bcast().
func (r *Latch) Clear() {
	r.mut.Lock()
	r.drain()
	r.avail = false
	r.mut.Unlock()
}
~~~