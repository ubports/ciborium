/*
 * Copyright 2015 Canonical Ltd.
 *
 * Authors:
 * Manuel de la Pena : manuel.delapena@cannical.com
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * ciborium is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package udisks2

import (
	"launchpad.net/go-dbus/v1"
	. "launchpad.net/gocheck"
)

type DispatcherTestSuite struct {
	d         *dispatcher
	completed chan bool
}

var _ = Suite(&DispatcherTestSuite{})

func (s *DispatcherTestSuite) SetUpTest(c *C) {
	jobs_ch := make(chan Event)
	additions_ch := make(chan Event)
	remove_ch := make(chan Event)
	s.d = &dispatcher{nil, nil, nil, jobs_ch, additions_ch, remove_ch}
	s.completed = make(chan bool)
}

func (s *DispatcherTestSuite) TestProcessAdditionsJob(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/3")
	props := make(map[string]VariantMap)
	interfaces := make([]string, 0, 0)
	event := Event{path, props, interfaces}
	// create a goroutine to test the event
	go func() {
		fwd_e := <-s.d.Jobs
		c.Assert(fwd_e.Path, Equals, path)
		s.completed <- true
	}()
	s.d.processAddition(event)
	<-s.completed
}

func (s *DispatcherTestSuite) TestProcessAdditionsDrive(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/block_devices/mmcblk1")
	props := make(map[string]VariantMap)
	interfaces := make([]string, 0, 0)
	event := Event{path, props, interfaces}

	// create a goroutine to test the event

	go func() {
		fwd_e := <-s.d.Additions
		c.Assert(fwd_e.Path, Equals, path)
		s.completed <- true
	}()
	s.d.processAddition(event)
	<-s.completed
}

func (s *DispatcherTestSuite) TestProcessRemovalJob(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/3")
	props := make(map[string]VariantMap)
	interfaces := make([]string, 0, 0)
	event := Event{path, props, interfaces}
	// create a goroutine to test the event
	go func() {
		fwd_e := <-s.d.Jobs
		c.Assert(fwd_e.Path, Equals, path)
		s.completed <- true
	}()
	s.d.processRemoval(event)
	<-s.completed
}

func (s *DispatcherTestSuite) TestProcessRemovalDrive(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/block_devices/mmcblk1")
	props := make(map[string]VariantMap)
	interfaces := make([]string, 0, 0)
	event := Event{path, props, interfaces}
	// create a goroutine to test the event
	go func() {
		fwd_e := <-s.d.Removals
		c.Assert(fwd_e.Path, Equals, path)
		s.completed <- true
	}()
	s.d.processRemoval(event)
	<-s.completed
}
