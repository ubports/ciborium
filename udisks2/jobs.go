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
	"log"
	"runtime"
	"sort"

	"launchpad.net/go-dbus/v1"
)

type job struct {
	Event        Event
	Operation    string
	Paths        []string
	WasCompleted bool
}

type jobManager struct {
	onGoingJobs     map[dbus.ObjectPath]job
	FormatEraseJobs chan job
	FormatMkfsJobs  chan job
	UnmountJobs     chan job
	MountJobs       chan job
}

func newJobManager(d *dispatcher) *jobManager {
	// listen to the diff job events and ensure that they are dealt with in the correct channel
	ongoing := make(map[dbus.ObjectPath]job)
	eraseChan := make(chan job)
	mkfsChan := make(chan job)
	unmountChan := make(chan job)
	mountChan := make(chan job)
	m := &jobManager{ongoing, eraseChan, mkfsChan, unmountChan, mountChan}
	runtime.SetFinalizer(m, cleanJobData)

	// create a go routine that will filter the diff jobs
	go func() {
		for {
			select {
			case e := <-d.Jobs:
				log.Println("New event", e.Path, "Properties:", e.Props, "Interfaces:", e.Interfaces)
				if e.isRemovalEvent() {
					log.Print("Is removal event")
					m.processRemovalEvent(e)
				} else {
					m.processAdditionEvent(e)
				}
			}
		}
		log.Print("Job manager routine done")
	}()
	return m
}

func (m *jobManager) processRemovalEvent(e Event) {
	log.Println("Deal with job event removal", e.Path, e.Interfaces)
	if job, ok := m.onGoingJobs[e.Path]; ok {
		// assert that we did loose the jobs interface, the dispatcher does sort the interfaces
		i := sort.SearchStrings(e.Interfaces, dbusJobInterface)
		if i != len(e.Interfaces) {
			log.Print("Job completed.")
			// complete event found
			job.WasCompleted = true

			if job.Operation == formatErase {
				log.Print("Sending completed erase job")
				m.FormatEraseJobs <- job
			}

			if job.Operation == formateMkfs {
				log.Print("Sending completed mkfs job")
				m.FormatMkfsJobs <- job
			}

			if job.Operation == unmountFs {
				log.Print("Sending completed unmount job")
				m.UnmountJobs <- job
			}

			if job.Operation == mountFs {
				log.Print("Sending complete mount job")
				m.MountJobs <- job
			}

			log.Print("Removed ongoing job for path", e.Path)
			delete(m.onGoingJobs, e.Path)
			return
		} else {
			log.Println("Ignoring event for path", e.Path, "because the job interface was not lost")
			return
		}
	} else {
		log.Println("Ignoring event for path", e.Path)
		return
	}
}

func (m *jobManager) processAdditionEvent(e Event) {
	j, ok := m.onGoingJobs[e.Path]
	if !ok {
		log.Println("Creating job for new path", e.Path, "details are", e)
		log.Println("New job operation", e.Props.jobOperation())
		operation := e.Props.jobOperation()
		var paths []string
		if e.Props.isMkfsFormatJob() || e.Props.isUnmountJob() || e.Props.isMountJob() {
			log.Print("Get paths from formatMkfs or unmountFs or mountFs event.")
			paths = e.Props.getFormattedPaths()
		}

		j = job{e, operation, paths, false}
		m.onGoingJobs[e.Path] = j
	} else {
		log.Print("Updating job for path ", e.Path)
		j.Event = e
		if e.Props.isEraseFormatJob() {
			j.Operation = formatErase
		}
		if e.Props.isMkfsFormatJob() {
			j.Operation = formateMkfs
			j.Paths = e.Props.getFormattedPaths()
		}
	}

	if j.Operation == formatErase {
		log.Print("Sending erase job from addition.")
		m.FormatEraseJobs <- j
	} else if j.Operation == formateMkfs {
		log.Print("Sending format job from addition.")
		m.FormatMkfsJobs <- j
	} else if j.Operation == unmountFs {
		log.Print("Sending nmount job from addition.")
		m.UnmountJobs <- j
	} else {
		log.Println("Ignoring job event with operation", j.Operation)
	}
}

func (m *jobManager) free() {
	close(m.FormatEraseJobs)
	close(m.FormatMkfsJobs)
}

func cleanJobData(m *jobManager) {
	m.free()
}
