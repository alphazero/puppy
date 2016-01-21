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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ----------------------------------------------------------------------
// command-line flags and configuration

var conf = struct {
	fname                             string
	trafficLimitLow, trafficLimitHigh uint
	statPeriodSec, alertPeriodMin     uint
	logJournalSize, alertsJournalSize uint
}{
	"", 100, 10000, 1, 5, 1024, 1024,
}

func init() {
	flag.StringVar(&conf.fname, "f", conf.fname, "apache log file")
	flag.UintVar(&conf.trafficLimitLow, "tmin", conf.trafficLimitLow, "traffic min threshold ")
	flag.UintVar(&conf.trafficLimitHigh, "tmax", conf.trafficLimitHigh, "traffic max threshold ")
	flag.UintVar(&conf.statPeriodSec, "s", conf.statPeriodSec, "stat snapshot period (sec)")
	flag.UintVar(&conf.alertPeriodMin, "a", conf.alertPeriodMin, "alerts check period (min)")
}

// ----------------------------------------------------------------------
// cleanup

func cleanup() {
	fmt.Println("DEBUG - cleanup - tooleh.go")
	restoreTerminal()
}

// ----------------------------------------------------------------------
// state

// maintain a sliding window of tail emits
var logJournal *ringBuffer

// maintain a sliding window of alerts notices.
var alertsJournal *ringBuffer

// maintain reference to last period's alert notice. Reminder that alert notices
// are in {alertRaised, alertRecovered}. This reference may be nil, indicating
// no standing (un-recovered) alert is in effect, keeping in mind that if an alert
// condition is not recovered in the current cycle, a new alert will be raised
// and referenced here.
var activeAlert *alert

// collect resource specific periodic. We'll accumulate data for each
// cycle (c.f. conf.statPeriodSec) and then compute the snapshot analysis.
var accessMetrics *metrics

// last snapshot's statistical analysis
var accessStatistic *statistic

// ----------------------------------------------------------------------
// process loop

func main() {

	var stat int
	var e error

	/* insure cleanup of terminal */
	defer func(stat0 *int, e0 *error) {
		cleanup()
		if *e0 != nil {
			log.Printf("%s\n", (*e0).Error())
		}
		log.Printf("exit - stat: %d\n", *stat0)
		os.Exit(*stat0)
	}(&stat, &e)

	/// config & setup /////////////////////////////////////////////////////

	flag.Parse()
	if conf.fname == "" {
		e = fmt.Errorf("log file name (option -f) is required.")
		stat = 6
		return
	}

	/* -- state objects */

	alertsJournal = newRingBuffer(conf.alertsJournalSize)
	logJournal = newRingBuffer(conf.logJournalSize)
	resolution := uint16(conf.alertPeriodMin / conf.statPeriodSec)
	accessMetrics, e = newMetrics(resolution)
	if e != nil {
		stat = 10 // TODO let's clean this up ..
		return
	}

	/* -- clocks --- */

	// stats timer
	stats_timer := time.NewTicker(time.Second * time.Duration(conf.statPeriodSec))
	defer stats_timer.Stop()

	// alert timer
	alert_timer := time.NewTicker(time.Minute * time.Duration(conf.alertPeriodMin))
	defer alert_timer.Stop()

	/* -- signals --- */

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Kill, os.Interrupt)
	defer close(interrupt) /* yes, pedantic */

	ttyevent := make(chan os.Signal, 1)
	signal.Notify(ttyevent, syscall.SIGWINCH)
	defer close(ttyevent)

	/* -- inputs --- */

	// tail process
	var tailproc *tailProc
	tailproc, e = tail(conf.fname)
	if e != nil {
		stat = 1
		return
	}
	// user input
	var ui <-chan uiEvent
	ui, e = uiEventPipe()
	if e != nil {
		stat = 2
		return
	}

	/// processing loop ////////////////////////////////////////////////////

	refreshDisplay(true)
	for {
		select {
		case <-stats_timer.C:
			// note: REVU comment of the function below addresses high
			// perofmrance concerns.
			accessStatistic = accessMetrics.takeSnapshot()
			refreshDisplay(false)
		case <-alert_timer.C:
			// TODO III - current-alert
			refreshDisplay(false)
		case event, ok := <-ui:
			if !ok {
				e = fmt.Errorf("ui events channel unepxectedly closed. will exit.")
				stat = 3
				tailproc.stop <- true
				return
			}
			switch {
			case event.is(viewStats, viewAlerts, viewLog, viewDebug):
				setView(event)
			case event.is(pageUp, pageDown):
				scrollView(event)
			case event.is(doQuit):
				tailproc.stop <- true
				return
			default:
				beep()
			}

		case line, ok := <-tailproc.out:
			if !ok {
				e = fmt.Errorf("err - tail stopped. (killed?)")
				stat = 4
				return
			}
			entry, err := parseW3cCommonLogFormat(line)
			if err != nil {
				e = fmt.Errorf("err - failed to parse tail out - %s\n", err.Error())
				stat = 5
				tailproc.stop <- true
				return
			}
			accessMetrics.Update(entry)
			logJournal.add(string(line)) // REVU: this optional feature is likely not worth the perf. hit.
			if currentView.id == logView {
				displayLog()
			}
		case sig := <-interrupt:
			if tailproc != nil {
				tailproc.cmd.Process.Kill()
			}
			signal.Stop(interrupt)
			selfSignal(sig)
			tailproc.stop <- true
			stat = 7
			return
		case _ = <-ttyevent:
			updateWinSize()
			refreshDisplay(true)
		}
	}
}

// we need to let the shell know that the processes was killed.
// so we reissue the signal (expected to be Interrupt or Kill)
// to self and this time we will not be catching it.
//
func selfSignal(s os.Signal) error {
	proc, e := os.FindProcess(os.Getpid())
	if e != nil {
		return e
	}
	return proc.Signal(s)
}
