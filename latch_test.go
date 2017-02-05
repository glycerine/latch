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
	latch.Close(aligator)

	for j := 0; j < 3; j++ {
		if j > 0 {
			latch.Refresh()
		}
		for i := 0; i < sz; i++ {
			select {
			case b := <-latch.Ch():
				if b != aligator {
					t.Fatal("Close(aligator) means aligator should always be read on the latch")
				}
			default:
				t.Fatal("latch is now closed, should have read back aligator. Refresh should have restocked us.")
			}
		}
	}

	// multiple closes are fine:
	crocadile := &Packet{Item: "lyle"}
	latch.Close(crocadile)

	for j := 0; j < 3; j++ {
		if j > 0 {
			latch.Refresh()
		}
		for i := 0; i < sz; i++ {
			select {
			case b := <-latch.Ch():
				if b != crocadile {
					t.Fatal("Close(crocadile) means crocadile should always be read on the latch")
				}
			default:
				t.Fatal("latch is now closed, should have read back crocadile. Refresh should have restocked us.")
			}
		}
	}

	// and after Open, we should block
	latch.Open()
	select {
	case <-latch.Ch():
		t.Fatal("Open() means she should have blocked.")
	default:
		// ok, good.
	}

}
