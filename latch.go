/*

the latch proposal (latch would be a new type in Go)
----------------------------------------------------

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

Note: It would be preferred to have an `open(latch)` syntax, but
that is deferred because it requires a new language keyword.

background
----------

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
*/
package latch

import (
	"sync"
	"time"
)

// Latch: rethinking channels
//
// What if we had a channel
//
// * that could be closed more than once.
//
// * that could be re-opened.
//
// * that could convey a default value
// once they are closed.
//
// * that didn't need a background
// goroutine to service it.
//
// To create a Latch, we will
// leverage the property of buffered channels:
//  even though they are buffered, they block
//  *until* they have something in them.
//
type Latch struct {
	sz     int
	mut    sync.Mutex
	cur    *Packet
	ch     chan *Packet
	closed bool // when closed==true, <- receives on Ch() will be given cur.

	fillerStop chan struct{}
}

// Packet conveys either a data Item,
// or an Err (or, possibly, both).
type Packet struct {
	Item interface{}
	Err  error
}

// NewLatch
func NewLatch(sz int) *Latch {
	return &Latch{
		ch: make(chan *Packet, sz),
		sz: sz,
	}
}

// returns a read-only channel. This is
// on purpose -- we want to prevent
// anyone from putting values
// into the channel by means other than
// calling Close().
func (r *Latch) Ch() <-chan *Packet {
	return r.ch
}

// clients should call Open(), not drain() directly.
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

// Close is like closing an electrical circuit;
// closing the circuit
// allows current (data) to flow. The
// opposite, Open, halts and blocks flow.
// The nature of the data that flows
// is copies of pak.
//
// Close can be called multiple times, with
// different values of pak. Each call will
// drain the ch channel of any prior data,
// any replace it will sz copies of pak.
//
func (r *Latch) Close(pak *Packet) {
	r.mut.Lock()
	r.cur = pak
	r.drain() // drop any old values.
	r.closed = true
	for i := 0; i < r.sz; i++ {
		r.ch <- r.cur
	}
	r.mut.Unlock()
}

// Refresh "tops-up" a closed channel. Since
// the channel is of finite size, and
// we don't want to waste a background
// goroutine (for speed and space), clients
// can regularly call Refresh to make sure
// an Close()-ed channel still has copies
// of data. Otherwise, after sz accesses,
// receivers on Ch() will block.
//
// If you want absolute correctness, and
// can't be bothered with invoking Refresh()
// regularly to service your closed channel,
// call BackgroundRefresher() once instead.
//
func (r *Latch) Refresh() {
	r.mut.Lock()
	if r.closed {
		for len(r.ch) < r.sz {
			r.ch <- r.cur
		}
	}
	r.mut.Unlock()
}

// BackgroundRefresher starts a goroutine
// that tops-up your Closed channel
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

// Open drains the latch. After
// we return, receivers on Ch()
// will block until somebody
// calls Close().
func (r *Latch) Open() {
	r.mut.Lock()
	r.drain()
	r.closed = false
	r.mut.Unlock()
}
