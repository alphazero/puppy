// friend

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

import (
	"bufio"
	"log"
	"os"
)

// General notes:
// Initial release of puppy is based on a straight forward data-flow
// paradigm and a reactive processing and display back-end. As such,
// user interaction is perceptual but not structural. This fact is
// underlined by the choice of 'display' rather than 'UI'.
//

type uiEvent byte

// ----------------------------------------------------------------------
// keystroke -> event mappings

func doQuit(e uiEvent) bool     { return e == 'q' || e == '\033' }
func viewStats(e uiEvent) bool  { return e == 's' || e == 'S' }
func viewAlerts(e uiEvent) bool { return e == 'a' || e == 'A' }
func viewLog(e uiEvent) bool    { return e == 'l' || e == 'L' }
func viewDebug(e uiEvent) bool  { return e == 'd' }
func pageUp(e uiEvent) bool     { return e == 'p' } /* prev */
func pageDown(e uiEvent) bool   { return e == 'n' } /* next */

// returns true if any of the provided comparators (e.g. doQuit())
// match the receiver.
//
// REVU: consider renaming to isIn or isAny for clear expression of the
//       semantics.
func (e uiEvent) is(comp ...func(uiEvent) bool) (yes bool) {
	if len(comp) == 0 {
		return false
	}
	yes = comp[0](e)
	for _, c0 := range comp[1:] {
		yes = c0(e) || yes
	}
	return
}

// ----------------------------------------------------------------------
// user input

// reads input from stdin, converts to a uiEvent, and emits on the
// returned channel.
//
// keeping things simple (and not bothering with tty raw mode), the user
// input listener assumes a single charachter command entry followed by
// CR. Validation is delegated to the basic event loop.
//
// life-cycle: listener will exit on stdin close. The ouput channel
// is also closed to notify the consumer.
func uiEventPipe() (<-chan uiEvent, error) {
	output := make(chan uiEvent)
	go func() {
		defer close(output)
		r := bufio.NewReader(os.Stdin)
		for {
			b, e := r.ReadBytes('\n')
			if e != nil {
				log.Printf("user input - %s\n", e.Error())
				return
			}
			output <- uiEvent(b[0])
		}
	}()

	return output, nil
}
