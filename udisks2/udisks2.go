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
type Interfaces []string

type Storage struct {
	Path  dbus.ObjectPath
	Props InterfacesAndProperties
}

type driveMap map[dbus.ObjectPath]InterfacesAndProperties

type UDisks2 struct {
	conn         *dbus.Connection
	validFS      sort.StringSlice
	DriveAdded   chan *Storage
	driveAdded   *dbus.SignalWatch
	DriveRemoved chan dbus.ObjectPath
	driveRemoved *dbus.SignalWatch
	drives       driveMap
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

func (u *UDisks2) connectToSignalInterfacesRemoved() (*dbus.SignalWatch, error) {
	return u.connectToSignal(dbusObject, dbusObjectManagerInterface, dbusRemovedSignal)
}

func (u *UDisks2) initInterfacesWatchChan() {
	go func() {
		defer close(u.DriveAdded)
		defer close(u.DriveRemoved)
		for {
			select {
			case msg := <-u.driveAdded.C:
				var event Storage
				if err := msg.Args(&event.Path, &event.Props); err != nil {
					log.Print(err)
					continue
				}
				if event.desiredAddEvent(u.validFS) {
					u.drives[event.Path] = event.Props
					u.DriveAdded <- &event
				}
			case msg := <-u.driveRemoved.C:
				var objectPath dbus.ObjectPath
				var interfaces Interfaces
				if err := msg.Args(&objectPath, &interfaces); err != nil {
					log.Print(err)
					continue
				}
				if _, ok := u.drives[objectPath]; !ok {
					log.Println("not concerned about event for", objectPath)
					continue
				}
				if interfaces.desiredRemoveEvent() {
					delete(u.drives, objectPath)
					u.DriveRemoved <- objectPath
				}
			}
		}
		log.Print("Shutting down InterfacesAdded channel")
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
		if s.desiredAddEvent(u.validFS) {
			u.DriveAdded <- &s
		}
	}
}

func NewStorageWatcher(conn *dbus.Connection, filesystems ...string) (u *UDisks2) {
	u = &UDisks2{
		conn:         conn,
		validFS:      sort.StringSlice(filesystems),
		DriveAdded:   make(chan *Storage),
		DriveRemoved: make(chan dbus.ObjectPath),
		drives:       make(driveMap),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func cleanDriveWatch(u *UDisks2) {
	log.Print("Cancelling Interfaces signal watch")
	u.driveAdded.Cancel()
	u.driveRemoved.Cancel()
}

func (u *UDisks2) Init() (err error) {
	if u.driveAdded, err = u.connectToSignalInterfacesAdded(); err != nil {
		return err
	}
	if u.driveRemoved, err = u.connectToSignalInterfacesRemoved(); err != nil {
		return err
	}
	u.initInterfacesWatchChan()
	return nil
}

func (iface Interfaces) desiredRemoveEvent() bool {
	for i := range iface {
		if iface[i] == dbusFilesystemInterface {
			return true
		}
	}
	return false
}

func (s *Storage) desiredAddEvent(validFS sort.StringSlice) bool {
	propFS, ok := s.Props[dbusFilesystemInterface]
	if !ok {
		return false
	}
	if mountpoints, ok := propFS["MountPoints"]; ok {
		if reflect.TypeOf(mountpoints.Value).Kind() != reflect.Slice {
			log.Println(s.Path, "does not hold a MountPoints slice")
			return false
		}
		if l := reflect.ValueOf(mountpoints.Value).Len(); l > 0 {
			log.Println(l, "previous mountpoint(s) found")
			return false
		}
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
