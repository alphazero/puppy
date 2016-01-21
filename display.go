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
	"log"
	"os"
	"time"
)

// display consists of n distinct 'views'. Views are simply rendered
// from current state. Each view maintains a scroll state.

func init() {
	if e := checkForTerminal(); e != nil {
		log.Fatal(e.Error())
	}
	updateWinSize()
}

// ----------------------------------------------------------------------
// display state

var rows, cols uint

func updateWinSize() (e error) {
	rows, cols, e = getWinsize(os.Stdin.Fd())
	return e
}

type viewId byte

const (
	statsView = iota
	alertsView
	logView
	debugView
)

type view struct {
	id   viewId
	page uint // scroll state
}

var currentView view

func setView(event uiEvent) (e error) {
	switch {
	case event.is(viewStats):
		currentView = view{statsView, 0}
	case event.is(viewAlerts):
		currentView = view{alertsView, 0}
	case event.is(viewLog):
		currentView = view{logView, 0}
	case event.is(viewDebug):
		currentView = view{debugView, 0}
	default:
		return fmt.Errorf("BUG - unknown uiEvent: %v", event)
	}
	cls()
	return displayView()
}

func displayView() (e error) {
	switch currentView.id {
	case statsView:
		e = displayStats()
	case alertsView:
		e = displayAlerts()
	case logView:
		e = displayLog()
	case debugView:
		e = displayDebug()
	}
	return e
}

func refreshDisplay(clearscreen bool) error {
	if clearscreen {
		cls()
	}
	return displayView()
}

// pages range from 0->n, with page 0 indicating the first page, however
// to confuse things, note that page-up scrolls to page 0, and page-dn
// to the last page n.
func scrollView(event uiEvent) error {
	switch {
	case event.is(pageUp):
		if currentView.page > 0 {
			currentView.page--
		}
	case event.is(pageDown):
		currentView.page++
	default:
		return fmt.Errorf("BUG - unknown uiEvent: %v", event)
	}
	return nil
}

// ----------------------------------------------------------------------
// views

func displayDebug() error {
	cls()
	ttycmd(HOME)
	stdViewHeader("debug", 6)

	row := uint(3)
	displayDatum0("conf", conf, row, 1)
	row++

	displayDatum("logJournal", logJournal.String(), row, 1)
	row++

	displayDatum("alertsJournal", alertsJournal.String(), row, 1)
	row++

	displayDatum0("metrics", accessMetrics, row, 1)
	row += 5

	displayDatum0("statistic", accessStatistic, row, 1)
	row++

	move(row, 1)
	for i, v := range accessMetrics.traffic.buf {
		fmt.Printf("[%d]: %v\n", i, v)
	}
	move(rows, cols)
	return nil
}

// alerts view
func displayAlerts() error {
	ttycmd(HOME)
	stdViewHeader("alerts", 1)

	move(rows, cols)
	return nil
}

func displayLog0() error {
	ttycmd(BOLD)
	entries := logJournal.last(rows - 3)

	fmt.Printf("rows: %d\n", rows)
	fmt.Printf("journal.buf.cap: %d\n", logJournal.cap)
	fmt.Printf("journal.buf.len: %d\n", len(logJournal.buf))
	fmt.Printf("journal.xof: %d\n", logJournal.xof)
	fmt.Printf("entries: %d\n", len(entries))

	for _, entry := range entries {
		fmt.Printf("%s\n", entry)
	}
	return nil
}

// log view
func displayLog() error {

	ttycmd(HOME)
	stdViewHeader("log", 4)

	entries := logJournal.last(rows - 3)

	/* view port */
	for n := uint(1); n <= uint(len(entries)); n++ {
		move(rows-n, 1)
		ttycmd(CLEARLINE)
		fmt.Printf("%s", entries[n-1])
		//	fmt.Printf("   %d", n)
	}
	ttycmd(NORMTEXT)

	move(rows, 1)
	ttycmd("[40;92;7m")
	fmt.Printf("sce-tail")
	ttycmd("[40;93;7m")
	ttycmd(NORMTEXT)
	ttyfmt(" My Love                                                          ", BOLD)

	move(rows, cols)
	return nil
}

func moveJustified(row uint, s string) {
	slen := uint(len(s))
	move(row, cols-slen+1)
}

func displayDatum0(label string, v interface{}, row, col uint) {
	displayDatum(label, fmt.Sprintf("%v", v), row, col)
}

func displayDatum(label, value string, row, col uint) {
	move(row, col)
	ttycmd(BOLD)
	fmt.Printf("%s", label)
	ttycmd(NORMTEXT)
	fmt.Printf(": %s", value)
}

func fillRow(row uint, c byte) {
	move(row, 1)
	for col := uint(0); col < cols; col++ {
		fmt.Printf("%c", c)
	}
}

func stdViewHeader(view string, color int) {
	tstr := time.Now().String()[:19]

	ttyfmt(view, BOLD, codefmt(BGCOLOR, 8), codefmt(FGCOLOR, color))
	move(1, 8)
	ttyfmt(conf.fname, BOLD)
	moveJustified(1, tstr)
	ttyfmt(tstr, BOLD, codefmt(FGCOLOR, 7))
	fillRow(2, '-')
}

// stats-view
func displayStats() error {
	ttycmd(HOME)
	stdViewHeader("stats", 5)

	//	fillRow(2, '-')
	displayDatum("requests", "1023034801", 3, 1)
	displayDatum("GET", "78.9%", 3, 24)
	displayDatum("PUT", "78.9%", 3, 36)
	displayDatum("POST", "78.9%", 3, 48)
	displayDatum("DEL", "78.9%", 3, 61)
	displayDatum("resources", "15", 4, 1)
	displayDatum("top-resource", "/wiki", 4, 24)
	displayDatum("users", "15", 5, 1)
	displayDatum("top-user", "alphazero", 5, 24)
	displayDatum("hosts", "9", 6, 1)
	displayDatum("top-host", "192.232.1.3", 6, 24)
	fillRow(7, '-')
	move(8, 1)
	ttyfmt("req %%", BOLD, UNDERLINE)
	move(8, 10)
	ttyfmt("req cnt", BOLD, UNDERLINE)
	move(8, 21)
	ttyfmt("resource", BOLD, UNDERLINE)

	/* view port */
	t0 := time.Now().UnixNano()
	for n := uint(9); n <= rows-2; n++ {
		move(n, 1)
		ttycmd(CLEARLINE)
		ttycmd(NORMTEXT)
		if time.Now().UnixNano()%0x7 == 0 {
			fgcolor(1)
		}
		fmt.Printf("%d | %2d", t0+int64(n), n)
	}
	ttycmd(NORMTEXT)

	move(rows, 1)
	ttycmd("[40;92;7m")
	fmt.Printf("sce-tail")
	ttycmd("[40;93;7m")
	ttycmd(NORMTEXT)
	ttyfmt(" My Love                                                          ", BOLD)

	move(rows, cols)
	return nil
}
