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
	"sort"

	"launchpad.net/go-dbus/v1"
	. "launchpad.net/gocheck"
)

type InterfacesAndPropertiesTestSuite struct {
	properties InterfacesAndProperties
}

var _ = Suite(&InterfacesAndPropertiesTestSuite{})

func (s *InterfacesAndPropertiesTestSuite) SetUpTest(c *C) {
	s.properties = make(map[string]VariantMap)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMountedMissingInterface(c *C) {
	// empty properties means that the interface is missing
	c.Assert(s.properties.isMounted(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMountedMissingMountPoints(c *C) {
	// add the expected interface but without the mount points property
	s.properties[dbusFilesystemInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.isMounted(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMountedNotSlize(c *C) {
	s.properties[dbusFilesystemInterface] = make(map[string]dbus.Variant)
	s.properties[dbusFilesystemInterface]["MountPoints"] = dbus.Variant{5}
	c.Assert(s.properties.isMounted(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMountedZeroMountPoints(c *C) {
	mount_points := make([]string, 0, 0)
	s.properties[dbusFilesystemInterface] = make(map[string]dbus.Variant)
	s.properties[dbusFilesystemInterface]["MountPoints"] = dbus.Variant{mount_points}
	c.Assert(s.properties.isMounted(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMountedSeveralMountPoints(c *C) {
	mount_points := make([]string, 1, 1)
	mount_points[0] = "/random/mount/point"
	s.properties[dbusFilesystemInterface] = make(map[string]dbus.Variant)
	s.properties[dbusFilesystemInterface]["MountPoints"] = dbus.Variant{mount_points}
	c.Assert(s.properties.isMounted(), Equals, true)
}

func (s *InterfacesAndPropertiesTestSuite) TestHasPartitionMissingInterface(c *C) {
	// an empty map should result in false
	c.Assert(s.properties.hasPartition(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestHasPartitionMissinUUID(c *C) {
	// add the interface with no properties
	s.properties[dbusPartitionInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.hasPartition(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestHasParitionMissingTable(c *C) {
	s.properties[dbusPartitionInterface] = make(map[string]dbus.Variant)
	s.properties[dbusPartitionInterface]["UUID"] = dbus.Variant{"A UUID"}
	c.Assert(s.properties.hasPartition(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestHasParitionPresent(c *C) {
	s.properties[dbusPartitionInterface] = make(map[string]dbus.Variant)
	s.properties[dbusPartitionInterface]["UUID"] = dbus.Variant{"A UUID"}
	s.properties[dbusPartitionInterface]["Table"] = dbus.Variant{"A Table"}
	c.Assert(s.properties.hasPartition(), Equals, true)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsPartitionableMissingInterface(c *C) {
	// an empty map should result in false
	c.Assert(s.properties.isPartitionable(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsParitionableMissingHint(c *C) {
	s.properties[dbusBlockInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.isPartitionable(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsParitionableHintNotBool(c *C) {
	s.properties[dbusBlockInterface] = make(map[string]dbus.Variant)
	s.properties[dbusBlockInterface]["HintPartitionable"] = dbus.Variant{"A String"}
	c.Assert(s.properties.isPartitionable(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsPartitionable(c *C) {
	s.properties[dbusBlockInterface] = make(map[string]dbus.Variant)
	s.properties[dbusBlockInterface]["HintPartitionable"] = dbus.Variant{true}
	c.Assert(s.properties.isPartitionable(), Equals, true)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsNotPartitionable(c *C) {
	s.properties[dbusBlockInterface] = make(map[string]dbus.Variant)
	s.properties[dbusBlockInterface]["HintPartitionable"] = dbus.Variant{false}
	c.Assert(s.properties.isPartitionable(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsEraseFormatJobMissingInterface(c *C) {
	// an empty map should result in false
	c.Assert(s.properties.isEraseFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsEraseFormatJobMissingOperation(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.isEraseFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsEraseFormatJobWrongType(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{false}
	c.Assert(s.properties.isEraseFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsEraseFormatJobWrongOperation(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{"false"}
	c.Assert(s.properties.isEraseFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsEraseFormatJob(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{"format-erase"}
	c.Assert(s.properties.isEraseFormatJob(), Equals, true)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMkfsFormatJobMissingInterface(c *C) {
	c.Assert(s.properties.isMkfsFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMkfsFormatJobMissingOperation(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.isMkfsFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMkfsFormatJobWrongType(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{true}
	c.Assert(s.properties.isMkfsFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMkfsFormatJobWrongOperation(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{"false"}
	c.Assert(s.properties.isMkfsFormatJob(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsMkfsFormatJob(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{"format-mkfs"}
	c.Assert(s.properties.isMkfsFormatJob(), Equals, true)
}

func (s *InterfacesAndPropertiesTestSuite) TestGetFormattedPathsMissingInterface(c *C) {
	paths := s.properties.getFormattedPaths()
	c.Assert(len(paths), Equals, 0)
}

func (s *InterfacesAndPropertiesTestSuite) TestGetFormattedPathsMissingProperty(c *C) {
	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	paths := s.properties.getFormattedPaths()
	c.Assert(len(paths), Equals, 0)
}

func (s *InterfacesAndPropertiesTestSuite) TestGetFormattedPaths(c *C) {
	firstPath := "/path/to/new/fs/1"
	secondPath := "/path/to/new/fs/2"
	thirdPath := "/path/to/new/fs/3"

	objsPaths := make([]interface{}, 3, 3)
	objsPaths[0] = firstPath
	objsPaths[1] = secondPath
	objsPaths[2] = thirdPath

	s.properties[dbusJobInterface] = make(map[string]dbus.Variant)
	s.properties[dbusJobInterface]["Operation"] = dbus.Variant{"format-mkfs"}
	s.properties[dbusJobInterface]["Objects"] = dbus.Variant{objsPaths}

	paths := s.properties.getFormattedPaths()
	//sort.Strings(paths)

	c.Assert(len(paths), Equals, len(objsPaths))
	c.Assert(sort.SearchStrings(paths, firstPath), Not(Equals), len(paths))
	c.Assert(sort.SearchStrings(paths, secondPath), Not(Equals), len(paths))
	c.Assert(sort.SearchStrings(paths, thirdPath), Not(Equals), len(paths))
}

func (s *InterfacesAndPropertiesTestSuite) TestIsFileSystemNoInterface(c *C) {
	c.Assert(s.properties.isFilesystem(), Equals, false)
}

func (s *InterfacesAndPropertiesTestSuite) TestIsFileSystem(c *C) {
	s.properties[dbusFilesystemInterface] = make(map[string]dbus.Variant)
	c.Assert(s.properties.isFilesystem(), Equals, true)
}
