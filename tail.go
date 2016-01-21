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
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

// General note:
// For the intitial release, the target platforms are *nixes and it is expected
// that the host provides the canonical 'tail' command. Certainly it is possible
// to replicate tail -F directly, but that will substantially add to the complexity
// of the initial release.

// encapsulates the required bits to manage and interact with tail process
// launched by the main puppy process.
type tailProc struct {
	cmd  *exec.Cmd
	out  <-chan []byte
	stop chan<- bool
}

// REVU: a knob to twist to possibly remedy the impedence
//       of the logView (which kills throughput) when active,
//       but we want this as small as possible (> 0) to maintain
//       termporal fidelity with the actual log events. The
//       higher the number, the greater the time lag between log file
//       timestamps and their bubbling up in the stats view.
const tailoutChanSize = 1

// launches tail <fname> with -F option (follow rollover/truncation).
// tail stderr is piped to the main process's. tail output is stripped
// of the terminal CR/LF and forwarded via the tailProc.out channel.
//
// The goroutines of this function will close both channels in the
// returned tailProc on exit.
//
// Known BUG: it has proven difficult to consistently shutdown the
// tail process if the main puppy process is SIGKILLed. REVU: let's
// try executing the kill command directly via exec.Cmd as the issue
// may be related to Go runtime and how it behaves when killed.
//
func tail(fname string) (*tailProc, error) {
	withError := func(fmtstr string, args ...interface{}) (*tailProc, error) {
		return nil, fmt.Errorf(fmtstr, args...)
	}
	if fname == "" {
		return withError("tail - fname is nil")
	}
	finfo, e := os.Stat(fname)
	if e != nil {
		return withError(fmt.Sprintf("ERR - tail - %s", e.Error()))
	} else if finfo.IsDir() {
		return withError(fmt.Sprintf("ERR - tail - %s is a directory", fname))
	}

	tailcmd := exec.Command("tail", "-F", fname)
	tailout, e := tailcmd.StdoutPipe()
	if e != nil {
		return withError("tail - cmd.StdoutPipe - %s", e.Error())
	}
	tailerr, e := tailcmd.StderrPipe()
	if e != nil {
		return withError("tail - cmd.StderrPipe - %s", e.Error())
	}
	if e = tailcmd.Start(); e != nil {
		return withError("tail - cmd.Start - %s", e.Error())
	}

	shutdown := make(chan bool, 1)
	go func() {
		<-shutdown
		if e := tailcmd.Process.Kill(); e != nil {
			log.Printf("error - tail.process.kill - e:%s", e.Error())
		}
		close(shutdown)
	}()

	output := make(chan []byte, tailoutChanSize)
	go func() {
		defer close(output)
		r := bufio.NewReader(tailout)
		for {
			line, e := r.ReadBytes('\n')
			switch e {
			case nil:
				output <- line[:len(line)-1]
			case io.EOF:
				return
			default:
				log.Printf("tail-reader - %s\n", e.Error())
				return
			}
		}
	}()

	go func() {
		r := bufio.NewReader(tailerr)
		for {
			line, e := r.ReadBytes('\n')
			switch e {
			case nil:
				log.Printf("tail-errout - %s\n", line[:len(line)-1])
			case io.EOF:
				return
			default:
				log.Printf("tail-errout - %s\n", e.Error())
				return
			}
		}
	}()

	return &tailProc{tailcmd, output, shutdown}, nil
}
