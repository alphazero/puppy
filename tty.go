//friend

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
	"os/exec"
	"syscall"
	"unsafe"
)

// General note:
// Again, per the general notes on 'tail' process, various matters
// are taken for granted given the assumption that the runtime host
// is a *nix and 'echo', 'stty', etc. are available.

// set on init - used on shutdown
var stty_restore string

// sets up the attached terminal for puppy's use. Any error here is
// treated as fatal and will os.Exit without ceremony.
//
func init() { // get tty current settings
	ttystate, e := sttycmd("-g")
	if e != nil {
		log.Fatalf("err - stty -g;  %s\n", e.Error())
	}
	stty_restore = string(ttystate[:len(ttystate)-1])

	// turn off terminal echo
	_, e = sttycmd("-echo")
	if e != nil {
		log.Fatalf("err - stty -echo - %s\n", e.Error())
	}
}

// puppy is not head-less and requires an attached terminal.
func checkForTerminal() (e error) {
	check := func(e error, fd uintptr, stream string) error {
		if e != nil {
			return e
		}
		r, c, e := getWinsize(fd)
		if e != nil || (r == 0 && c == 0) {
			return fmt.Errorf("ERR - fatal - %s stream is not attached to a terminal", stream)
		}
		return nil
	}
	e = check(e, os.Stdin.Fd(), "input")
	e = check(e, os.Stdout.Fd(), "output")
	return
}

// complement to init(), a somewhat vigorous attempt to restore terminal
// state. Per various (adhoc/functional) tests of various failure/stop
// modes of puppy (e.g. on quit or on interrupts) this is fine and will
// not leave the terminal in a messed up state. That said, this needs
// *nix graybeard code review.
func restoreTerminal() {
	var e error
	cls()
	// restore terminal
	// an error here would be un-expected as program cleanup
	// is only called after successful init, i.e. an attached tty,
	// so we'll treat the errors as unexpected faults.
	_, e = sttycmd("echo")
	if e != nil {
		log.Fatalf("fault - cleanup - stty echo - %s\n", e.Error())
	}

	_, e = sttycmd(stty_restore)
	if e != nil {
		log.Fatalf("fault - cleanup - stty <restore> - %s\n", e.Error())
	}

	ttycmd("[0m")
}

/////////////////////////////////////////////////////////////////////////
/// TTY terminal control ////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////

// General note:
// Basic printf to stdout provides quite a lot of the required functions
// but we still require use of system calls (for window size) and stty
// for setting params.

// ----------------------------------------------------------------------
// ioctl syscalls

type winsize struct{ rows, cols, xpixel, ypixel uint16 }

func getWinsize(fd uintptr) (rows, cols uint, err error) {
	var ws winsize
	res, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)))
	if res < 0 {
		return 0, 0, fmt.Errorf("ERR - getWinsize - errno:%d\n", errno)
	}
	return uint(ws.rows), uint(ws.cols), nil
}

// ----------------------------------------------------------------------
// escape route

type code string

const (
	HOME          = code("[H")
	UP            = code("[%dA")
	DOWN          = code("[%dB")
	RIGHT         = code("[%dC")
	LEFT          = code("[%dD")
	MOVE          = code("[%d;%dH")
	FGCOLOR       = code("[3%dm")
	BGCOLOR       = code("[4%dm")
	CLEARSCREEN   = code("[2J")
	CLEARLINE     = code("[2K")
	SAVECSTATE    = code("7")
	RESTORECSTATE = code("8")
	NORMTEXT      = code("[0m")
	BOLD          = code("[1m")
	DIM           = code("[2m")
	UNDERLINE     = code("[4m")
	BLINK         = code("[5m")
	REVERSE       = code("[7m")
	INVISIBLE     = code("[8m")
)

func codefmt(c code, args ...interface{}) code {
	return code(fmt.Sprintf(string(c), args...))
}

func ttycmd(c code)                       { fmt.Printf("\033%s", c) }
func ttycmdf(c code, args ...interface{}) { ttycmd(codefmt(c, args...)) }
func ttycmds(c ...code) {
	for _, c0 := range c {
		fmt.Printf("\033%s", c0)
	}
}

/* convenience */

func cls()               { ttycmds(HOME, CLEARSCREEN) }
func beep()              { fmt.Printf("\007") }
func move(row, col uint) { ttycmdf(MOVE, row, col) }
func fgcolor(col uint)   { ttycmdf(FGCOLOR, col) }
func bgcolor(col uint)   { ttycmdf(FGCOLOR, col) }

func moveJustified(row uint, s string) {
	slen := uint(len(s))
	move(row, cols-slen+1)
}

func fillRow(row uint, c byte) {
	move(row, 1)
	for col := uint(0); col < cols; col++ {
		fmt.Printf("%c", c)
	}
}

// text is always restored to NORMTEXT on return
func ttyfmt(s string, codes ...code) {
	ttycmds(codes...)
	fmt.Printf(s)
	ttycmd(NORMTEXT)
}

// ----------------------------------------------------------------------
// direct stty

// this is likely the slowest method of directign the TTY but it is only
// used on startup and shutdown.
func sttycmd(args ...string) (out []byte, e error) {
	stty := exec.Command("stty", args...)
	stty.Stdin = os.Stdin
	return stty.Output()
}
