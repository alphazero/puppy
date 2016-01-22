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
	"net/url"
)

// nod toward possible multi-format extensions
// not used.
type parseFn func([]byte) (map[string]string, error)

// structure captures the basic w3c Common Log Format.
type logEntry struct {
	remoteHost string
	rfc931     string
	user       string
	date       string
	tmz        string
	method     string
	address    string
	protocol   string
	status     uint
	bytes      uint
	uri        *url.URL
}

func (p *logEntry) section() string {
	path := p.uri.Path
	if len(path) > 2 {
		for i, b := range []byte(path)[1:] {
			if b == '/' {
				return path[:i+1]
			}
		}
	}
	return path
}

// function attempts parse of provided line.
// 'entry' is always nil in case of errors.
// 'entry' is nil if line is a comment (in which case error will be nil).
func parseW3cCommonLogFormat(line []byte) (entry *logEntry, err error) {
	if len(line) == 0 { /* ignore but possible bug */
		err = fmt.Errorf("err - parseW3cCommonLogFormat - unexpected zero-len input")
		return
	}
	if line[0] == '#' { /* ignore log directives & meta-data for now */
		return
	}
	entry = &logEntry{}
	n, e := fmt.Sscanf(string(line), "%s %s %s [%s %5s] \"%s %s %8s\" %d %d",
		&entry.remoteHost,
		&entry.rfc931,
		&entry.user,
		&entry.date,
		&entry.tmz,
		&entry.method,
		&entry.address,
		&entry.protocol,
		&entry.status,
		&entry.bytes)
	if e != nil {
		err = fmt.Errorf("ERR - parseW3cCommonLogFormat - n:%d e:%s\n", n, e.Error())
		return
	}
	/* verify here */
	entry.uri, e = url.Parse(entry.address)
	if e != nil {
		err = fmt.Errorf("ERR - parseW3cCommonLogFormat - url.Parse - e:%s\n", e.Error())
		return
	}

	return
}
