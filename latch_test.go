package latch

import "testing"

func TestLatch(t *testing.T) {

	sz := 3
	latch := NewLatch(sz)

	select {
	case <-latch.Ch():
		t.Fatal("latch starts open; it should have blocked")
	default:
		// ok, good.
	}

	aligator := &Packet{Item: "bill"}
	latch.Bcast(aligator)

	for j := 0; j < 3; j++ {
		if j > 0 {
			latch.Refresh()
		}
		for i := 0; i < sz; i++ {
			select {
			case b := <-latch.Ch():
				if b != aligator {
					t.Fatal("Bcast(aligator) means aligator should always be read on the latch")
				}
			default:
				t.Fatal("latch is now closed, should have read back aligator. Refresh should have restocked us.")
			}
		}
	}

	// multiple closes are fine:
	crocadile := &Packet{Item: "lyle"}
	latch.Bcast(crocadile)

	for j := 0; j < 3; j++ {
		if j > 0 {
			latch.Refresh()
		}
		for i := 0; i < sz; i++ {
			select {
			case b := <-latch.Ch():
				if b != crocadile {
					t.Fatal("Bcast(crocadile) means crocadile should always be read on the latch")
				}
			default:
				t.Fatal("latch is now closed, should have read back crocadile. Refresh should have restocked us.")
			}
		}
	}

	// and after Clear, we should block
	latch.Clear()
	select {
	case <-latch.Ch():
		t.Fatal("Clear() means recevie should have blocked.")
	default:
		// ok, good.
	}

}
