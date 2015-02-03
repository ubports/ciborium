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
	"strings"

	"launchpad.net/go-dbus/v1"
)

const (
	jobPrefixPath = "/org/freedesktop/UDisks2/jobs/"
	blockDevices  = "/org/freedesktop/UDisks2/block_devices/"
)

type Interfaces []string

type Event struct {
	Path       dbus.ObjectPath
	Props      InterfacesAndProperties
	Interfaces Interfaces
}

func (e *Event) isRemovalEvent() bool {
	return len(e.Interfaces) != 0
}

type dispatcher struct {
	conn           *dbus.Connection
	additionsWatch *dbus.SignalWatch
	removalsWatch  *dbus.SignalWatch
	Jobs           chan Event
	Additions      chan Event
	Removals       chan Event
}

func connectToSignal(conn *dbus.Connection, path dbus.ObjectPath, inter, member string) (*dbus.SignalWatch, error) {
	w, err := conn.WatchSignal(&dbus.MatchRule{
		Type:      dbus.TypeSignal,
		Sender:    dbusName,
		Interface: dbusObjectManagerInterface,
		Member:    member,
		Path:      path})
	return w, err
}

func newDispatcher(conn *dbus.Connection) (*dispatcher, error) {
	// connect to the reuired signals, if it is not possible, return nil
	add_w, err := connectToSignal(conn, dbusObject, dbusObjectManagerInterface, dbusAddedSignal)
	if err != nil {
		return nil, err
	}

	remove_w, err := connectToSignal(conn, dbusObject, dbusObjectManagerInterface, dbusRemovedSignal)
	if err != nil {
		return nil, err
	}

	jobs_ch := make(chan Event)
	additions_ch := make(chan Event)
	remove_ch := make(chan Event)

	d := &dispatcher{conn, add_w, remove_w, jobs_ch, additions_ch, remove_ch}
	runtime.SetFinalizer(d, cleanDispatcherData)

	// create the go routines used to grab the events and dispatch them accordingly
	go func() {
		for msg := range d.additionsWatch.C {
			var event Event
			if err := msg.Args(&event.Path, &event.Props); err != nil {
				log.Print(err)
				continue
			}
			d.processAddition(event)
		}
	}()

	go func() {
		for msg := range d.removalsWatch.C {
			var event Event
			if err := msg.Args(&event.Path, &event.Interfaces); err != nil {
				log.Print(err)
				continue
			}
			sort.Strings(event.Interfaces)
			d.processRemoval(event)
		}
	}()

	return d, nil
}

func (d *dispatcher) free() {
	// cancel all watches so that goroutines are done and close the
	// channels
	d.additionsWatch.Cancel()
	d.removalsWatch.Cancel()
	close(d.Jobs)
	close(d.Additions)
	close(d.Removals)
}

func (d *dispatcher) processAddition(event Event) {
	log.Print("Dealing with an add event from path ", event.Path)
	// according to the object path we know if the even was a job one or not
	if strings.HasPrefix(string(event.Path), jobPrefixPath) {
		d.Jobs <- event
	}
	if strings.HasPrefix(string(event.Path), blockDevices) {
		d.Additions <- event
	}
}

func (d *dispatcher) processRemoval(event Event) {
	log.Print("Dealing with a remove event from path ", event.Path)
	// according to the object path we know if the even was a job one or not
	if strings.HasPrefix(string(event.Path), jobPrefixPath) {
		d.Jobs <- event
	}
	if strings.HasPrefix(string(event.Path), blockDevices) {
		d.Removals <- event
	}
}

func cleanDispatcherData(d *dispatcher) {
	d.free()
}
