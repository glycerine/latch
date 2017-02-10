package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {

	b := NewBcast(3)
	go func() {
		for {
			fmt.Printf("top of receive loop\n")
			select {
			case v := <-b.Ch:
				b.Reload() // always do this after a receive on Ch, to enable broadcast.
				fmt.Printf("received on Ch: %v\n", v)
			}
		}
	}()

	b.Set(4)
	time.Sleep(20 * time.Millisecond)
	b.Set(5)
	time.Sleep(20 * time.Millisecond)
	b.Off()
	time.Sleep(20 * time.Millisecond)
}

type Bcast struct {
	Ch  chan int
	mu  sync.Mutex
	on  bool
	cur int
}

func NewBcast(expectedDiameter int) *Bcast {
	return &Bcast{
		Ch: make(chan int, expectedDiameter+1),
	}
}

func (b *Bcast) On() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on = true
	b.fill()
}

func (b *Bcast) Set(val int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on = true
	b.cur = val
	b.drain()
	b.fill()
}

func (b *Bcast) Off() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on = false
	b.drain()
}

// drain all messages, leaving Ch empty.
func (b *Bcast) drain() {
	// empty chan
	for {
		select {
		case <-b.Ch:
		default:
			return
		}
	}
}

// all clients should call Reload after receiving
// on the channel. This makes such channels
// self-servicing.
func (b *Bcast) Reload() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.on {
		b.fill()
	}
}

// fill up the channel
func (b *Bcast) fill() {
	for {
		select {
		case b.Ch <- b.cur:
			fmt.Printf("filled once with %v\n", b.cur)
		default:
			return
		}
	}
}
