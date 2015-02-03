/*
 * Copyright 2014 Canonical Ltd.
 *
 * Authors:
 * Manuel de la Pena : manuel.delapena@cannical.com
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * nuntium is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package udisks2

import (
	"log"
	"runtime"
	"sort"

	"launchpad.net/go-dbus/v1"
)

type job struct {
	Operation    string
	Paths        []string
	WasCompleted bool
}

type jobManager struct {
	onGoingJobs     map[dbus.ObjectPath]job
	FormatEraseJobs chan job
	FormatMkfsJobs  chan job
}

func newJobManager(d *dispatcher) *jobManager {
	// listen to the diff job events and ensure that they are dealt with in the correct channel
	ongoing := make(map[dbus.ObjectPath]job)
	erase_ch := make(chan job)
	mkfs_ch := make(chan job)
	m := &jobManager{ongoing, erase_ch, mkfs_ch}
	runtime.SetFinalizer(m, cleanJobData)

	// create a go routine that will filter the diff jobs
	go func() {
		for e := range d.Jobs {
			if e.isRemovalEvent() {
				m.processRemovalEvent(e)
			} else {
				m.processAdditionEvent(e)
			}
		}
	}()
	return m
}

func (m *jobManager) processRemovalEvent(e Event) {
	job, ok := m.onGoingJobs[e.Path]
	if ok {
		// assert that we did loose the jobs interface, the dispatcher does sort the interfaces
		i := sort.SearchStrings(e.Interfaces, dbusJobInterface)
		if i != len(e.Interfaces) {
			// complete event found
			job.WasCompleted = true

			if job.Operation == formatErase {
				log.Print("Sending new erase job")
				m.FormatEraseJobs <- job
			}

			if job.Operation == formateMkfs {
				log.Print("Sending mkfs job")
				m.FormatMkfsJobs <- job
			}

			log.Print("Removed ongoing job for path", e.Path)
			delete(m.onGoingJobs, e.Path)
			return
		} else {
			log.Print("Ignoring event for path ", e.Path, " because the job interface was not lost")
			return
		}
	} else {
		log.Print("Ignoring event for path ", e.Path)
		return
	}
}

func (m *jobManager) processAdditionEvent(e Event) {
	j, ok := m.onGoingJobs[e.Path]
	if !ok {
		log.Print("Creating job for new path ", e.Path)
		var operation string
		var paths []string
		if e.Props.isEraseFormatJob() {
			operation = formatErase
		}
		if e.Props.isMkfsFormatJob() {
			operation = formateMkfs
			paths = e.Props.getFormattedPaths()
		}

		j = job{operation, paths, false}
		m.onGoingJobs[e.Path] = j
	} else {
		log.Print("Updating job for path ", e.Path)
		if e.Props.isEraseFormatJob() {
			j.Operation = formatErase
		}
		if e.Props.isMkfsFormatJob() {
			j.Operation = formateMkfs
			j.Paths = e.Props.getFormattedPaths()
		}
	}

	if j.Operation == formatErase {
		m.FormatEraseJobs <- j
	}

	if j.Operation == formateMkfs {
		m.FormatMkfsJobs <- j
	}
}

func (m *jobManager) free() {
	close(m.FormatEraseJobs)
	close(m.FormatMkfsJobs)
}

func cleanJobData(m *jobManager) {
	m.free()
}
