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
	"fmt"
	"time"
)

// REVU: possibly better named as alertNoticeType.
//       c.f. alert struct
type alertType string

const (
	alertRaised    = "alert-raised"
	alertRecovered = "alert-recovered"
)

// note: rolls over at bounds (minimally ~91 days given a 1m alert period)
var nextId uint16

func nextAlertId() (id uint16) {
	id = nextId
	nextId++
	return
}

// REVU: possibly better named as alertNotice.
//
// structure encapsulates a notice regarding an alert (raised or recovered).
// Note that the associated pair (rasied, recovered) share the same id.
// For globally unique identifiers, we can mux id & timestamp (which will be
// unique).
type alert struct {
	id        uint16
	typ       alertType
	timestamp time.Time
	msg       string
}

// REVU: possibly better to include the typ and id
//       given that msg is directly accessible.
func (p *alert) String() string { return p.msg }

// creates a new alert-raised (notice). Returns error on zero-value ts.
func newAlert(reqcnt uint, ts time.Time) (*alert, error) {

	if ts.IsZero() {
		return nil, fmt.Errorf("bug - newAlert - assert - timestamp is zero-value")
	}
	id := nextAlertId()
	fmtstr := "High traffic alert {%d} - hits = {%d}, triggered at {%s}"
	msg := fmt.Sprintf(fmtstr, id, reqcnt, ts.Format(time.RFC3339))
	return &alert{id, alertRaised, ts, msg}, nil
}

// creates the alert-recovered (notice) complement for the reciever.
// Returns error if receiver is not of type 'alertRaised', or if
// input arg is zero-value.
func (p *alert) recovered(ts time.Time) (*alert, error) {
	if p.typ == alertRecovered {
		return nil, fmt.Errorf("bug - alert.recovered - invalid - receiver is not a raised alert")
	}
	if ts.IsZero() {
		return nil, fmt.Errorf("bug - alert.recovered - assert - timestamp is zero-value")
	}
	fmtstr := "Alert {%d} recovered at {%s}"
	msg := fmt.Sprintf(fmtstr, p.id, ts.Format(time.RFC3339)) // REVU: may want to use same format for view headers.
	return &alert{p.id, alertRecovered, ts, msg}, nil
}
