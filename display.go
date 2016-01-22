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

// General notes:
//
// display consists of n distinct 'views'. Views are simply rendered
// from current state. Each view maintains a scroll state.
//
// 'display' and 'views' should not be confused with MVC, as the paradigm
// is more along the lines of IPD (Input > Process > Display) to keep
// things simple.
//
// Note that display functions are called by the main routine and are
// not intended for concurrent use. Should a later release add additional
// output back-ends (e.g. a network endpoint), either a channel based
// event model needs to be used, or the puppy model needs to become
// concurrent (which is probably not a good idea ;)

func init() {
	if e := checkForTerminal(); e != nil {
		log.Fatal(e.Error())
	}
	updateWinSize()
}

// ----------------------------------------------------------------------
// display state

// set by updateWinSize (typically to respond to signal.Notify events)
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

// TODO: clean this up or retire it
func displayDebug() error {
	cls()
	ttycmd(HOME)
	stdViewHeader("debug", 6)

	row := uint(3)
	displayDatum0("conf", conf, row, 1)
	row++

	displayDatum0("metrics", accessMetrics, row, 1)
	row += 5

	stats := accessStatistic
	if stats != nil {
		displayDatum0("statistic", stats, row, 1)
		row++
		displayDatum0("tot", stats.accessCnt.total, row, 1)
		row++
		displayDatum0("counts", stats.accessCnt, row, 1)
		row++
		displayDatum0("ratios", stats.accessRatio, row, 1)
		row++
	}
	move(row, 1)
	for i, v := range accessMetrics.traffic.buf {
		fmt.Printf("[%d]: %v\n", i, v)
	}
	move(rows, cols)
	return nil
}

// stats-view
func displayStats() error {
	ttycmds(HOME, CLEARSCREEN)
	stdViewHeader("stats", 5)

	/// snapshot head-ups summary ///////////////////////////////////////

	stats := accessStatistic
	if stats == nil { // TODO: init accessStatistic with zval and remove this check
		return nil
	}
	// percent formatter - ##.# precision is sufficient
	pfmtr := func(v float64) string {
		return fmt.Sprintf("%03.1f%%", v*100.)
	}
	/* traffic summary */
	displayDatum0("requests", stats.accessCnt.total, 3, 1)
	displayDatum("GET", pfmtr(stats.accessRatio.gets), 3, 24)
	displayDatum("PUT", pfmtr(stats.accessRatio.puts), 3, 36)
	displayDatum("POST", pfmtr(stats.accessRatio.posts), 3, 48)
	displayDatum("DEL", pfmtr(stats.accessRatio.dels), 3, 61)
	displayDatum("OTHER", pfmtr(stats.accessRatio.other), 3, 73)
	/* aggregate and specific active resource, user, and host */
	displayDatum0("resources", stats.byResource.total, 4, 1)
	displayDatum("top-resource", stats.byResource.top, 4, 24)
	displayDatum0("users", stats.byUser.total, 5, 1)
	displayDatum("top-user", stats.byUser.top, 5, 24)
	displayDatum0("hosts", stats.byHost.total, 6, 1)
	displayDatum("top-host", stats.byHost.top, 6, 24)

	fillRow(7, '-') /* REVU: let's go fully reto and draw lines */

	/// access by attribute /////////////////////////////////////////////

	// REVU TODO tri-state flag in {resource, user, host} with default
	//      TODO in which case factor our the generic table renderer
	/* table header */
	move(8, 1)
	ttyfmt("req %%", BOLD, UNDERLINE)
	move(8, 10)
	ttyfmt("req cnt", BOLD, UNDERLINE)
	move(8, 21)
	ttyfmt("resource", BOLD, UNDERLINE)

	// view data
	inOrder := stats.byResource.inOrder

	/* view port */
	cnt := uint(len(inOrder))
	xof := cnt - 1
	sak := uint(9) // scroll adjust faktor
	viewportLim := rows - sak
	lim := min(viewportLim, cnt)
	for n := uint(0); n < lim; n++ {
		move(n+sak, 1)
		ttycmd(CLEARLINE)
		ttycmd(NORMTEXT)
		item := inOrder[xof-n]
		itemTotal := item.counter.total
		itemRatio := float64(itemTotal) / float64(stats.accessCnt.total)
		fmt.Printf("%5s  %9d    %s", pfmtr(itemRatio), itemTotal, item.name)
	}

	/* standard footer */
	stdViewFooter()
	return nil
}

// alerts view
func displayAlerts() error {
	ttycmd(HOME)
	stdViewHeader("alerts", 1)

	viewportLim := rows - 3
	entries := alertsJournal.last(viewportLim)
	cnt := uint(len(entries))
	sak := viewportLim - cnt // scroll adjust factor
	/* view port */
	for n := uint(1); n <= cnt; n++ {
		move(rows-n-sak, 1)
		ttycmd(CLEARLINE)
		alrt := entries[n-1]
		if n == 1 && alrt == activeAlert {
			fgcolor(1)
		}
		fmt.Printf("%s", alrt)
		ttycmd(NORMTEXT)
	}

	stdViewFooter()
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
	}
	ttycmd(NORMTEXT)

	stdViewFooter()
	return nil
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

func stdViewFooter() {
	move(rows, 1)
	ttycmd(codefmt(FGCOLOR, 3))
	ttycmd(codefmt(BGCOLOR, 7))
	fillRow(rows, ' ')
	move(rows, 1)

	// active alert status must be always visible
	switch {
	case activeAlert == nil:
		ttyfmt(" RELAX ", BOLD, codefmt(BGCOLOR, 2), codefmt(FGCOLOR, 7))
	case activeAlert.typ == alertRaised:
		ttyfmt(" ALERT ", BOLD, codefmt(BGCOLOR, 1), codefmt(FGCOLOR, 8))
		ttycmd(codefmt(FGCOLOR, 0))
		ttycmd(codefmt(BGCOLOR, 7))
		ttycmd(BOLD)
		lim := min(uint(len(activeAlert.String())), cols-9)
		fmt.Printf(" %s", activeAlert.String()[:lim])
	case activeAlert.typ == alertRecovered:
		ttyfmt(" RECOV ", BOLD, codefmt(BGCOLOR, 5), codefmt(FGCOLOR, 7))
		ttycmd(codefmt(FGCOLOR, 0))
		ttycmd(codefmt(BGCOLOR, 7))
		ttycmd(BOLD)
		lim := min(uint(len(activeAlert.String())), cols-9)
		fmt.Printf(" %s", activeAlert.String()[:lim])
	}
	move(rows, cols)
	ttycmd(NORMTEXT)
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

func min(a, b uint) uint {
	if a > b {
		return b
	}
	return a
}
