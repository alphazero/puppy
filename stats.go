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
	"sort"
	"time"
)

// General notes: this initial design of the stateful bits of puppy hasn't
// been profiled but it is likely that any focus on performance should
// start here. this initial version is aimed at simply providing the required
// feature set for what is effectively puppy's stateful model.

/// basic counts ////////////////////////////////////////////////////////

// REVU: not strictly necessary. Nod towards future extensibility
type AccessMetrics interface {
	Update(*logEntry) error // TODO rename to accessLog
}

// ---------------------------------------------------------------------
// access info

type accessCounter struct {
	total, gets, puts, posts, dels, other uint
}
type accessRatio struct {
	gets, puts, posts, dels, other float64
}

func (p *accessCounter) ratios() *accessRatio {
	ratios := &accessRatio{}
	if p.total > 0 {
		n := float64(p.total)
		ratios.gets = float64(p.gets) / n
		ratios.puts = float64(p.puts) / n
		ratios.dels = float64(p.dels) / n
		ratios.posts = float64(p.posts) / n
		ratios.other = float64(p.other) / n
	}
	return ratios
}
func (p *accessCounter) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - accessCounter.update - assert - access is nil")
	}
	switch access.method {
	case "GET":
		p.gets++
	case "PUT":
		p.puts++
	case "POST":
		p.posts++
	case "DEL":
		p.dels++
	default:
		p.other++
	}
	p.total++
	return nil
}

type namedCounter struct {
	name    string
	counter *accessCounter
}
type ByTotal []namedCounter

func (a ByTotal) Len() int           { return len(a) }
func (a ByTotal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTotal) Less(i, j int) bool { return a[i].counter.total < a[j].counter.total }

// ---------------------------------------------------------------------
// measures

// meaures captures distinct data views on access information.
type measures struct {
	resources map[string]*accessCounter
	users     map[string]*accessCounter
	hosts     map[string]*accessCounter
}

func newMeasures() *measures {
	p := &measures{
		make(map[string]*accessCounter),
		make(map[string]*accessCounter),
		make(map[string]*accessCounter),
	}
	return p
}
func (p *measures) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - measures.update - assert - access is nil")
	}
	keys := []string{access.section(), access.remoteHost, access.user}
	maps := []map[string]*accessCounter{p.resources, p.hosts, p.users}
	for i, key := range keys {
		info, ok := (maps[i])[key]
		if !ok {
			info = &accessCounter{}
			(maps[i])[key] = info
		}
		info.Update(access) // ok to ignore error here
	}
	return nil
}

// used to compute elements for overall traffic metrics
func (p *measures) summarize() *accessCounter {
	summary := &accessCounter{}
	for _, entry := range p.resources {
		summary.gets += entry.gets
		summary.puts += entry.puts
		summary.posts += entry.posts
		summary.dels += entry.dels
		summary.other += entry.other
		summary.total += entry.total
	}
	return summary
}

// panics
func (p *measures) statsBy(attribute string) *accessStats {
	var data map[string]*accessCounter
	//	var column []namedCounter
	var stats accessStats

	switch attribute {
	case "user":
		data = p.users
	case "host":
		data = p.hosts
	case "resource":
		data = p.resources
	default:
		panic(fmt.Sprintf("bug - measures.statsBy - unknown attribute %s", attribute))
	}
	stats.total = uint(len(data))
	stats.inOrder = make([]namedCounter, stats.total) // make it regardless of len
	if stats.total == 0 {
		return &stats
	}

	i := 0
	for key, counter := range data {
		stats.inOrder[i] = namedCounter{key, counter}
		i++
	}
	sort.Sort(ByTotal(stats.inOrder))
	stats.top = stats.inOrder[len(data)-1].name
	stats.topRatio = float64(stats.inOrder[len(data)-1].counter.total) / float64(stats.total)

	return &stats
}

// ---------------------------------------------------------------------
// metrics

// metrics capture the overall high level view on the data collected.
// On every snapshot period, the wip is finalized as 'snapshot', with
// associated addition of a new traffic element.
type metrics struct {
	traffic     *ringBuffer // <*accessCounter> : accumulated periodic data
	snapshot    *measures   // immutable snapshot of last period's measure
	snapshot_ts time.Time   // timestamp of snapshot update
	wip         *measures   // in-progress measures of current period
}

type accessStats struct {
	total    uint
	top      string
	topRatio float64
	inOrder  []namedCounter
}

// REVU: for an extensible variant of puppy, use a map[key]value.
type statistic struct {
	// access counts and ratio breakdown by access method
	accessCnt   *accessCounter
	accessRatio *accessRatio

	// stats for various access attributes
	byResource *accessStats
	byUser     *accessStats
	byHost     *accessStats
}

// limit rsolution to a reasonable 2^16 - 1.
func newMetrics(resolution uint16) (*metrics, error) {
	if resolution == 0 {
		return nil, fmt.Errorf("err - initStats - resolution must be non-zero")
	}
	s := &metrics{}
	s.traffic = newRingBuffer(uint(resolution))
	s.snapshot = newMeasures()
	s.wip = newMeasures()

	return s, nil
}

func (p *metrics) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - metrics.update - assert - access is nil")
	}
	return p.wip.Update(access)
}

// called periodically to take snapshot of running measures and
// update the overall traffic metrics. this function will panic on detected
// bugs.
//
// REVU: for concurrent version, async hand-off in conjunction with addition of
//       a serialization point (e.g. mutext) would address requirements for
//       a high performance version. It is not strictly necessary to return a
//       future in that case but obviously no longer returning a statistic ref.
//
func (p *metrics) takeSnapshot() *statistic {

	// update metrics with collected data in wip
	p.snapshot = p.wip
	p.snapshot_ts = time.Now()
	p.wip = newMeasures()
	accessCnt := p.snapshot.summarize()
	p.traffic.add(accessCnt)

	// compute the stats for the snapshot
	//
	// in the simple case this boils down to sorting the
	// access info  uri and other attributes (in this case user and host).
	stats := &statistic{}
	stats.accessCnt = accessCnt
	stats.accessRatio = accessCnt.ratios()

	//	stats.byResource.total = uint(len(p.snapshot.resources))
	stats.byResource = p.snapshot.statsBy("resource")
	stats.byUser = p.snapshot.statsBy("user")
	stats.byHost = p.snapshot.statsBy("host")

	// traffic data in general

	// sort by resource address and compute
	return stats
}

func (p *metrics) String() string {
	return fmt.Sprintf("metrics\n\t%s\n\t%v\n\t%v", p.traffic, p.snapshot, p.wip)
}
