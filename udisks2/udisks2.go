/*
 * Copyright 2014 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@cannical.com
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
	"reflect"
	"runtime"
	"sort"

	"log"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName                   = "org.freedesktop.UDisks2"
	dbusObject                 = "/org/freedesktop/UDisks2"
	dbusObjectManagerInterface = "org.freedesktop.DBus.ObjectManager"
	dbusBlockInterface         = "org.freedesktop.UDisks2.Block"
	dbusFilesystemInterface    = "org.freedesktop.UDisks2.Filesystem"
	dbusAddedSignal            = "InterfacesAdded"
	dbusRemovedSignal          = "InterfacesRemoved"
)

type VariantMap map[string]dbus.Variant
type InterfacesAndProperties map[string]VariantMap

type Storage struct {
	Path  dbus.ObjectPath
	Props InterfacesAndProperties
}

type driveMap map[dbus.ObjectPath]InterfacesAndProperties

type UDisks2 struct {
	conn       *dbus.Connection
	validFS    sort.StringSlice
	DriveAdded chan *Storage
	driveAdded *dbus.SignalWatch
	drives     driveMap
}

func (u *UDisks2) connectToSignal(path dbus.ObjectPath, inter, member string) (*dbus.SignalWatch, error) {
	w, err := u.conn.WatchSignal(&dbus.MatchRule{
		Type:      dbus.TypeSignal,
		Sender:    dbusName,
		Interface: dbusObjectManagerInterface,
		Member:    member,
		Path:      path})
	return w, err
}

func (u *UDisks2) connectToSignalInterfacesAdded() (*dbus.SignalWatch, error) {
	return u.connectToSignal(dbusObject, dbusObjectManagerInterface, dbusAddedSignal)
}

func (u *UDisks2) initInterfacesAddedChan() {
	go func() {
		for msg := range u.driveAdded.C {
			var addedEvent Storage
			if err := msg.Args(&addedEvent.Path, &addedEvent.Props); err != nil {
				log.Print(err)
				continue
			}
			if addedEvent.desiredEvent(u.validFS) {
				u.DriveAdded <- &addedEvent
			}
		}
		log.Print("Shutting down InterfacesAdded channel")
		close(u.DriveAdded)
	}()

	u.emitExistingDevices()
}

func (u *UDisks2) emitExistingDevices() {
	obj := u.conn.Object(dbusName, dbusObject)
	reply, err := obj.Call(dbusObjectManagerInterface, "GetManagedObjects")
	if err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}

	allDevices := make(map[dbus.ObjectPath]InterfacesAndProperties)
	if err := reply.Args(&allDevices); err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}

	for objectPath, props := range allDevices {
		s := Storage{objectPath, props}
		if s.desiredEvent(u.validFS) {
			u.DriveAdded <- &s
		}
	}
}

func NewStorageWatcher(conn *dbus.Connection, filesystems ...string) (u *UDisks2) {
	u = &UDisks2{
		conn:       conn,
		validFS:    sort.StringSlice(filesystems),
		DriveAdded: make(chan *Storage),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func cleanDriveWatch(u *UDisks2) {
	log.Print("Cancelling InterfacesAdded signal watch")
	u.driveAdded.Cancel()
}

func (u *UDisks2) Init() (err error) {
	if u.driveAdded, err = u.connectToSignalInterfacesAdded(); err != nil {
		return err
	}
	u.initInterfacesAddedChan()
	return nil
}

func (s *Storage) desiredEvent(validFS sort.StringSlice) bool {
	propFS, ok := s.Props[dbusFilesystemInterface]
	if !ok {
		return false
	}
	if _, ok := propFS["MountPoints"]; ok {
		// already mounted
		return false
	}

	propBlock, ok := s.Props[dbusBlockInterface]
	if !ok {
		return false
	}
	id, ok := propBlock["IdType"]
	if !ok {
		log.Println(s.Path, "doesn't hold IdType")
		return false
	}

	fs := reflect.ValueOf(id.Value).String()
	i := validFS.Search(fs)
	if i >= validFS.Len() || validFS[i] != fs {
		log.Println(fs, "not in:", validFS, "for", s.Path)
		return false
	}

	return true
}

func (s *Storage) Mount(conn *dbus.Connection) (mountpoint string, err error) {
	obj := conn.Object(dbusName, s.Path)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	reply, err := obj.Call(dbusFilesystemInterface, "Mount", options)
	if err != nil {
		return "", err
	}
	if err := reply.Args(&mountpoint); err != nil {
		return "", err
	}

	return mountpoint, err
}
