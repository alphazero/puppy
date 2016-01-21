//    Copyright © 2016 Joubin Houshyar. All rights reserved.
//
//    This file is part of puppy.
//
//    puppy is free software: you can redistribute it and/or modify
//    it under the terms of the GNU Affero General Public License as
//    published by the Free Software Foundation, either version 3 of
//    the License, or (at your option) any later version.
//
//    puppy is distributed in the hope that it will be useful,
//    but WITHOUT ANY WARRANTY; without even the implied warranty of
//    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//    GNU Affero General Public License for more details.
//
//    You should have received a copy of the GNU Affero General Public
//    License along with puppy.  If not, see <http://www.gnu.org/licenses/>.

package main

import "fmt"

// General note:
// this needs to be profiled as it is consistently used in the main
// loop.

// a very basic ring buffer with log/journal semantics. We'll
// use this to maintain a journal of log-entries and raised alerts.
type ringBuffer struct {
	buf []interface{}
	xof uint
	cap uint // len() redundant; for convenience
}

func (r *ringBuffer) String() string {
	return fmt.Sprintf("ringBuffer - buf.len:%d - xof:%d", len(r.buf), r.xof)
}

func newRingBuffer(cap uint) *ringBuffer {
	return &ringBuffer{make([]interface{}, cap), 0, cap}
}

// add item to the buffer. nil input is ignored.
func (r *ringBuffer) add(item interface{}) {
	if item == nil {
		return
	}
	r.buf[r.xof] = item
	r.xof = (r.xof + 1) % r.cap
	return
}

// return (up to max) last (FILO) entries
func (r *ringBuffer) last(max uint) []interface{} {
	if max > r.cap {
		max = r.cap
	}
	arr := make([]interface{}, max)
	xof := uint(0)
	if r.xof > 0 {
		for i := int(r.xof - 1); xof < max && i >= 0; i-- {
			if r.buf[i] == nil {
				break
			}
			arr[xof] = r.buf[i]
			xof++
		}
	}
	for i := int(r.cap - 1); xof < max && i >= int(r.xof); i-- {
		if r.buf[i] == nil {
			break
		}
		arr[xof] = r.buf[i]
		xof++
	}
	return arr[:xof]
}

// returns all items in FILO order
func (r *ringBuffer) items() []interface{} {
	return r.last(r.cap)
}

// removes all items and resets ringBuffer state
func (r *ringBuffer) clear() {
	for i := 0; i < len(r.buf); i++ {
		r.buf[i] = nil
	}
	r.xof = 0
}

/*
func main() {
	traffic := newRingBuffer(8)
	fmt.Printf("Salaam!\n")

	fmt.Printf("%s - %v\n", traffic, traffic.items())
	traffic.add("1")
	traffic.add("2")
	traffic.add(nil)
	fmt.Printf("%s - %v\n", traffic, traffic.items())
	traffic.clear()
	traffic.add("a")
	fmt.Printf("%s - %v\n", traffic, traffic.items())
	traffic.clear()
	for i := 0; i < 27; i++ {
		traffic.add(i)
	}
	fmt.Printf("%s - %v\n", traffic, traffic.items())
	for n := uint(0); n <= traffic.cap; n++ {
		fmt.Printf("%s - %v\n", traffic, traffic.last(n))
	}
}
*/
