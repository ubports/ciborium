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

type JobManagerTestSuite struct {
	ongoing   map[dbus.ObjectPath]job
	events    chan Event
	manager   *jobManager
	completed chan bool
}

var _ = Suite(&JobManagerTestSuite{})

func (s *JobManagerTestSuite) SetUpTest(c *C) {
	s.ongoing = make(map[dbus.ObjectPath]job)
	eraseChan := make(chan job)
	mkfsChan := make(chan job)
	unmountChan := make(chan job)
	mountChan := make(chan job)

	s.manager = &jobManager{s.ongoing, eraseChan, mkfsChan, unmountChan, mountChan}
	s.completed = make(chan bool)
}

func (s *JobManagerTestSuite) TearDownTest(c *C) {
	close(s.manager.FormatEraseJobs)
	close(s.manager.FormatMkfsJobs)
	close(s.completed)
}

func (s *JobManagerTestSuite) TestProcessAddEventNewErase(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/3")

	go func() {
		for j := range s.manager.FormatEraseJobs {
			c.Assert(j.Operation, Equals, formatErase)
			c.Assert(j.WasCompleted, Equals, false)
			// assert that the job is present in the ongoing map
			_, ok := s.ongoing[path]
			c.Assert(ok, Equals, true)
			s.completed <- true
		}
	}()

	interfaces := make([]string, 0, 0)

	props := make(map[string]VariantMap)
	props[dbusJobInterface] = make(map[string]dbus.Variant)
	props[dbusJobInterface][operationProperty] = dbus.Variant{formatErase}
	objsPaths := make([]string, 1, 1)
	objsPaths[0] = "/path/to/erased/fs"
	props[dbusJobInterface][objectsProperty] = dbus.Variant{objsPaths}

	event := Event{path, props, interfaces}
	s.manager.processAdditionEvent(event)
	<-s.completed
}

func (s *JobManagerTestSuite) TestProcessAddEventNewFormat(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/1")

	go func() {
		for j := range s.manager.FormatMkfsJobs {
			c.Assert(j.Operation, Equals, formateMkfs)
			c.Assert(j.WasCompleted, Equals, false)
			// assert that the job is present in the ongoing map
			_, ok := s.ongoing[path]
			c.Assert(ok, Equals, true)
			s.completed <- true
		}
	}()

	interfaces := make([]string, 0, 0)

	props := make(map[string]VariantMap)
	props[dbusJobInterface] = make(map[string]dbus.Variant)
	props[dbusJobInterface][operationProperty] = dbus.Variant{formateMkfs}
	objsPaths := make([]interface{}, 1, 1)
	objsPaths[0] = "/path/to/new/fs"
	props[dbusJobInterface][objectsProperty] = dbus.Variant{objsPaths}

	event := Event{path, props, interfaces}
	s.manager.processAdditionEvent(event)
	<-s.completed
}

func (s *JobManagerTestSuite) TestProcessAddEventPresent(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/1")

	// add a ongoing job for the given path
	s.ongoing[path] = job{}

	go func() {
		for j := range s.manager.FormatMkfsJobs {
			c.Assert(j.Operation, Equals, formateMkfs)
			c.Assert(j.WasCompleted, Equals, false)
			// assert that the job is present in the ongoing map
			_, ok := s.ongoing[path]
			c.Assert(ok, Equals, true)
			s.completed <- true
		}
	}()

	interfaces := make([]string, 0, 0)

	props := make(map[string]VariantMap)
	props[dbusJobInterface] = make(map[string]dbus.Variant)
	props[dbusJobInterface][operationProperty] = dbus.Variant{formateMkfs}
	objsPaths := make([]interface{}, 1, 1)
	objsPaths[0] = "/path/to/new/fs"
	props[dbusJobInterface][objectsProperty] = dbus.Variant{objsPaths}

	event := Event{path, props, interfaces}
	s.manager.processAdditionEvent(event)
	<-s.completed
}

func (s *JobManagerTestSuite) TestProcessRemovalEventMissing(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/1")
	interfaces := make([]string, 1, 1)
	interfaces[0] = dbusJobInterface
	props := make(map[string]VariantMap)

	event := Event{path, props, interfaces}
	// nothing bad should happen
	s.manager.processRemovalEvent(event)

}

func (s *JobManagerTestSuite) TestProcessRemovalEventInterfaceMissing(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/1")
	interfaces := make([]string, 1, 1)
	interfaces[0] = "com.test.Random"
	props := make(map[string]VariantMap)

	event := Event{path, props, interfaces}

	s.ongoing[path] = job{}

	// nothing bad should happen
	s.manager.processRemovalEvent(event)
}

func (s *JobManagerTestSuite) TestProcessRemovalEventMkfs(c *C) {
	path := dbus.ObjectPath("/org/freedesktop/UDisks2/jobs/1")

	// create an erase job and add it to the ongoing map, check that the job
	// is fwd to the channel as completed and removed from the map
	formattedPaths := make([]string, 1, 1)
	formattedPaths[0] = "/one/path/to/a/fmormatted/fs/1"
	presentJob := job{Event{}, formateMkfs, formattedPaths, false}

	s.ongoing[path] = presentJob

	go func() {
		for j := range s.manager.FormatMkfsJobs {
			c.Assert(j.Operation, Equals, formateMkfs)
			c.Assert(j.WasCompleted, Equals, true)
			c.Assert(len(j.Paths), Equals, 1)
			s.completed <- true
		}
	}()

	interfaces := make([]string, 1, 1)
	interfaces[0] = dbusJobInterface
	props := make(map[string]VariantMap)

	event := Event{path, props, interfaces}

	s.manager.processRemovalEvent(event)
	<-s.completed
}
