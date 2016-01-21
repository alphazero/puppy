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
)

/// basic counts ////////////////////////////////////////////////////////

// REVU: not strictly necessary. Nod towards future extensibility
type AccessMetrics interface {
	Update(*logEntry) error // TODO rename to accessLog
}

// ---------------------------------------------------------------------
// access info

type accessInfo struct {
	gets, puts, posts, dels, other uint
}

func (p *accessInfo) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - accessInfo.update - assert - access is nil")
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
	return nil
}

// ---------------------------------------------------------------------
// measures

// meaures captures distinct data views on access information.
type measures struct {
	// in progress stats by resource (URI)
	resources map[string]*accessInfo
	users     map[string]*accessInfo
	hosts     map[string]*accessInfo
}

func newMeasures() *measures {
	p := &measures{
		make(map[string]*accessInfo),
		make(map[string]*accessInfo),
		make(map[string]*accessInfo),
	}
	return p
}
func (p *measures) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - measures.update - assert - access is nil")
	}
	keys := []string{access.section(), access.uri.Host, access.user}
	maps := []map[string]*accessInfo{p.resources, p.hosts, p.users}
	for i, key := range keys {
		info, ok := (maps[i])[key]
		if !ok {
			info = &accessInfo{}
			(maps[i])[key] = info
		}
		info.Update(access) // ok to ignore error here
	}
	return nil
}

// used to compute elements for overall traffic metrics
func (p *measures) summarize() *accessInfo {
	summary := &accessInfo{}
	for _, entry := range p.resources {
		summary.gets += entry.gets
		summary.puts += entry.puts
		summary.posts += entry.posts
		summary.dels += entry.dels
		summary.other += entry.other
	}
	return summary
}

// ---------------------------------------------------------------------
// metrics

// metrics capture the overall high level view on the data collected.
// On every snapshot period, the wip is finalized as 'snapshot', with
// associated addition of a new traffic element.
type metrics struct {
	traffic  *ringBuffer // <*accessInfo> : accumulated periodic data
	snapshot *measures   // immutable snapshot of last period's measure
	wip      *measures   // in-progress measures of current period
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

func (p *metrics) String() string {
	return fmt.Sprintf("metrics\n\t%s\n\t%v\n\t%v", p.traffic, p.snapshot, p.wip)
}
func (p *metrics) Update(access *logEntry) error {
	if access == nil {
		return fmt.Errorf("err - metrics.update - assert - access is nil")
	}
	return p.wip.Update(access)
}

// called periodically to take snapshot of running measures and
// update the overall traffic metrics.
func (p *metrics) takeSnapshot() *metrics {
	p.snapshot = p.wip
	p.wip = newMeasures()
	p.traffic.add(p.snapshot.summarize())

	return p
}

func (p *metrics) analysis() *statistic {
	return nil
}

/*
type ByRequests []*requestStats

func (v ByRequests) Len() int {
	return len(v)
}
func (v ByRequests) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
func (v ByRequests) Less(i, j int) bool {
	return v[i].requests < v[j].requests
}

// Resource specific stats
type resourceStats struct {
	name  string
	stats requestStats
}

func (p resourceStats) update(requestLog *logEntry) {
	p.stats.requests++
	switch requestLog.method {
	case "GET":
		p.stats.gets++
	case "PUT":
		p.stats.puts++
	case "POST":
		p.stats.posts++
	case "DEL":
		p.stats.dels++
	default:
		p.stats.other++
	}
}

type resourceStatsMap map[string]*resourceStats
type accessStats struct {
	resources map[string]*resourceStats
	users     map[string]int
	hosts     map[string]int
}

func newResourceStatsMap() resourceStatsMap {
	rsm := make(map[string]*resourceStats)
	return rsm
}

func (p resourceStatsMap) update(requestLog *logEntry) {
	section := requestLog.section()
	resStats, ok := p[section]
	if !ok {
		resStats = &resourceStats{name: section}
		p[section] = resStats
	}
	resStats.update(requestLog)
}
*/
// REVU TODO traffic should use this
type statistic struct {
	total   accessInfo
	users   uint
	topUser string
	hosts   uint
	topHost string
	//	resources []*resourceStats // REVU: this data is already in metrics.snapshot
}

/*
func (p resourceStatsMap) analyze() *statsAnalysis {
	sa := &statsAnalysis{}

	sa.resources = make([]*resourceStats, len(p))
	//	users := make(map[string]int)
	//	hosts := make(map[string]int)
	var i int
	for id, rs := range p {
		sa.resources[i] = rs
		//		users[rs.user] = users[rs.user] + 1
		//		hosts[rs.host] = hosts[rs.host] + 1
		i++
	}
	// sorts by total request count
	sort.Sort(ByRequests(sa.resources))

	return sa
}
*/
