package clock

import (
	"testing"
	"time"
)

func TestWallClock(t *testing.T) {
	before := time.Now().Unix()
	n := Unix()
	after := time.Now().Unix()
	if n < before || n > after || n < 1700000000 {
		t.Fatal(n, before, after)
	}
}

func TestWallClockNano(t *testing.T) {
	before := time.Now().UnixNano()
	n := UnixNano()
	after := time.Now().UnixNano()
	if n < before || n > after || n < 1700000000*int64(time.Second) {
		t.Fatal(n, before, after)
	}
}
