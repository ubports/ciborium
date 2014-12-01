/*
 * Copyright 2014 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@canonical.com
 *
 * This file is part of ubuntu-emulator.
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * ubuntu-emulator is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"os"
	"path/filepath"
	"testing"

	. "launchpad.net/gocheck"
)

var _ = Suite(&StandardDirsTestSuite{})

type StandardDirsTestSuite struct {
	tmpDir string
}

func (s *StandardDirsTestSuite) SetUpTest(c *C) {
	s.tmpDir = c.MkDir()
}

func Test(t *testing.T) { TestingT(t) }

func (s *StandardDirsTestSuite) TestCreateFromScratch(c *C) {
	createStandardHomeDirs(s.tmpDir)

	for _, d := range []string{"Documents", "Downloads", "Music", "Pictures", "Videos"} {
		fi, err := os.Stat(filepath.Join(s.tmpDir, d))
		c.Assert(err, IsNil)
		c.Assert(fi.IsDir(), Equals, true)
	}
}

func (s *StandardDirsTestSuite) TestCreateWithPreExistingHead(c *C) {
	c.Assert(os.Mkdir(filepath.Join(s.tmpDir, "Documents"), 0755), IsNil)
	c.Assert(os.Mkdir(filepath.Join(s.tmpDir, "Downloads"), 0755), IsNil)

	c.Assert(createStandardHomeDirs(s.tmpDir), IsNil)

	for _, d := range []string{"Documents", "Downloads", "Music", "Pictures", "Videos"} {
		fi, err := os.Stat(filepath.Join(s.tmpDir, d))
		c.Assert(err, IsNil)
		c.Assert(fi.IsDir(), Equals, true)
	}
}

func (s *StandardDirsTestSuite) TestCreateWithPreExistingTail(c *C) {
	c.Assert(os.Mkdir(filepath.Join(s.tmpDir, "Videos"), 0755), IsNil)

	c.Assert(createStandardHomeDirs(s.tmpDir), IsNil)

	for _, d := range []string{"Documents", "Downloads", "Music", "Pictures", "Videos"} {
		fi, err := os.Stat(filepath.Join(s.tmpDir, d))
		c.Assert(err, IsNil)
		c.Assert(fi.IsDir(), Equals, true)
	}
}

func (s *StandardDirsTestSuite) TestCreateWithPreExistingMiddle(c *C) {
	c.Assert(os.Mkdir(filepath.Join(s.tmpDir, "Music"), 0755), IsNil)

	c.Assert(createStandardHomeDirs(s.tmpDir), IsNil)

	for _, d := range []string{"Documents", "Downloads", "Music", "Pictures", "Videos"} {
		fi, err := os.Stat(filepath.Join(s.tmpDir, d))
		c.Assert(err, IsNil)
		c.Assert(fi.IsDir(), Equals, true)
	}
}

func (s *StandardDirsTestSuite) TestCreateWithPreExistingNonDir(c *C) {
	musicFile := filepath.Join(s.tmpDir, "Music")
	f, err := os.Create(musicFile)
	c.Assert(err, IsNil)
	f.Close()

	c.Assert(createStandardHomeDirs(s.tmpDir), IsNil)

	fi, err := os.Stat(musicFile)
	c.Assert(err, IsNil)
	c.Assert(fi.IsDir(), Equals, false)

	for _, d := range []string{"Documents", "Downloads", "Pictures", "Videos"} {
		fi, err := os.Stat(filepath.Join(s.tmpDir, d))
		c.Assert(err, IsNil)
		c.Assert(fi.IsDir(), Equals, true)
	}
}
