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
	"errors"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"log"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName                   = "org.freedesktop.UDisks2"
	dbusObject                 = "/org/freedesktop/UDisks2"
	dbusObjectManagerInterface = "org.freedesktop.DBus.ObjectManager"
	dbusBlockInterface         = "org.freedesktop.UDisks2.Block"
	dbusDriveInterface         = "org.freedesktop.UDisks2.Drive"
	dbusFilesystemInterface    = "org.freedesktop.UDisks2.Filesystem"
	dbusAddedSignal            = "InterfacesAdded"
	dbusRemovedSignal          = "InterfacesRemoved"
)

type VariantMap map[string]dbus.Variant
type InterfacesAndProperties map[string]VariantMap
type Interfaces []string

type drive struct {
	path         dbus.ObjectPath
	blockDevices map[dbus.ObjectPath]InterfacesAndProperties
	driveInfo    InterfacesAndProperties
}

type driveMap map[dbus.ObjectPath]*drive

type Event struct {
	Path  dbus.ObjectPath
	Props InterfacesAndProperties
}

type mountpointMap map[dbus.ObjectPath]string

type UDisks2 struct {
	conn         *dbus.Connection
	validFS      sort.StringSlice
	DriveAdded   chan *Event
	driveAdded   *dbus.SignalWatch
	DriveRemoved chan dbus.ObjectPath
	driveRemoved *dbus.SignalWatch
	drives       driveMap
	mountpoints  mountpointMap
}

func NewStorageWatcher(conn *dbus.Connection, filesystems ...string) (u *UDisks2) {
	u = &UDisks2{
		conn:         conn,
		validFS:      sort.StringSlice(filesystems),
		DriveAdded:   make(chan *Event),
		DriveRemoved: make(chan dbus.ObjectPath),
		drives:       make(driveMap),
		mountpoints:  make(mountpointMap),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func (u *UDisks2) Mount(conn *dbus.Connection, s *Event) (mountpoint string, err error) {
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

	u.mountpoints[s.Path] = mountpoint
	return mountpoint, err
}

func (u *UDisks2) ExternalDrives() []drive {
	var drives []drive
	for _, d := range u.drives {
		if d.hasSystemBlockDevices() {
			continue
		}
		drives = append(drives, *d)
	}
	return drives
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

func (u *UDisks2) initInterfacesWatchChan() {
	go func() {
		defer close(u.DriveAdded)
		defer close(u.DriveRemoved)
		for {
			select {
			case msg := <-u.driveAdded.C:
				var event Event
				if err := msg.Args(&event.Path, &event.Props); err != nil {
					log.Print(err)
					continue
				}
				if err := u.processAddEvent(&event); err != nil {
					log.Print("Issues while processing ", event.Path, ": ", err)
				}
			case msg := <-u.driveRemoved.C:
				var objectPath dbus.ObjectPath
				var interfaces Interfaces
				if err := msg.Args(&objectPath, &interfaces); err != nil {
					log.Print(err)
					continue
				}
				if err := u.processRemoveEvent(objectPath, interfaces); err != nil {
					log.Println("Issues while processing remove event:", err)
				}
			}
		}
		log.Print("Shutting down InterfacesAdded channel")
	}()

	u.emitExistingDevices()
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
		s := &Event{objectPath, props}
		u.processAddEvent(s)
	}
}

func (u *UDisks2) processAddEvent(s *Event) error {
	if blockDevice, err := u.drives.addInterface(s); err != nil {
		return err
	} else if blockDevice {
		if u.desiredMountableEvent(s) {
			u.DriveAdded <- s
		}
	}

	return nil
}

func (u *UDisks2) processRemoveEvent(objectPath dbus.ObjectPath, interfaces Interfaces) error {
	mountpoint, mounted := u.mountpoints[objectPath]
	if mounted {
		log.Println("Removing mountpoint", mountpoint)
		delete(u.mountpoints, objectPath)
		if interfaces.desiredUnmountEvent() {
			u.DriveRemoved <- objectPath
		} else {
			return errors.New("mounted but does not remove filesystem interface")
		}
	}
	delete(u.drives, objectPath)
	return nil
}

func cleanDriveWatch(u *UDisks2) {
	log.Print("Cancelling Interfaces signal watch")
	u.driveAdded.Cancel()
	u.driveRemoved.Cancel()
}

func (iface Interfaces) desiredUnmountEvent() bool {
	for i := range iface {
		if iface[i] == dbusFilesystemInterface {
			return true
		}
	}
	return false
}

func (u *UDisks2) desiredMountableEvent(s *Event) bool {
	drivePath, err := s.getDrive()
	if err != nil {
		log.Println("Issues while getting drive:", err)
		return false
	}
	if ok := u.drives[drivePath].hasSystemBlockDevices(); ok {
		log.Println(drivePath, "which contains", s.Path, "has HintSystem set")
		return false
	}

	if drive, ok := u.drives[drivePath]; !ok {
		log.Println(drivePath, "not in drive map")
		return false
	} else {
		driveProps, ok := drive.driveInfo[dbusDriveInterface]
		if !ok {
			log.Println(drivePath, "doesn't hold a Drive interface")
			return false
		}
		if mediaRemovableVariant, ok := driveProps["MediaRemovable"]; !ok {
			log.Println(drivePath, "which holds", s.Path, "doesn't have MediaRemovable")
			return false
		} else {
			mediaRemovable := reflect.ValueOf(mediaRemovableVariant.Value).Bool()
			if !mediaRemovable {
				log.Println(drivePath, "which holds", s.Path, "is not MediaRemovable")
				return false
			}
		}
	}

	propFS, ok := s.Props[dbusFilesystemInterface]
	if !ok {
		return false
	}
	if mountpointsVariant, ok := propFS["MountPoints"]; ok {
		if reflect.TypeOf(mountpointsVariant.Value).Kind() != reflect.Slice {
			log.Println(s.Path, "does not hold a MountPoints slice")
			return false
		}
		if mountpoints := reflect.ValueOf(mountpointsVariant.Value).Len(); mountpoints > 0 {
			log.Println(mountpoints, "previous mountpoint(s) found")
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
	i := u.validFS.Search(fs)
	if i >= u.validFS.Len() || u.validFS[i] != fs {
		log.Println(fs, "not in:", u.validFS, "for", s.Path)
		return false
	}

	return true
}

func (d *drive) hasSystemBlockDevices() bool {
	for _, blockDevice := range d.blockDevices {
		propBlock, ok := blockDevice[dbusBlockInterface]
		if !ok {
			continue
		}
		if systemHintVariant, ok := propBlock["HintSystem"]; !ok {
			continue
		} else if systemHint := reflect.ValueOf(systemHintVariant.Value).Bool(); systemHint {
			return true
		}
	}
	return false
}

func (s *Event) getDrive() (dbus.ObjectPath, error) {
	propBlock, ok := s.Props[dbusBlockInterface]
	if !ok {
		return "", fmt.Errorf("interface %s not found", dbusBlockInterface)
	}
	driveVariant, ok := propBlock["Drive"]
	if !ok {
		return "", errors.New("property 'Drive' not found")
	}
	return dbus.ObjectPath(reflect.ValueOf(driveVariant.Value).String()), nil
}

func newDrive(s *Event) *drive {
	return &drive{
		path:         s.Path,
		blockDevices: make(map[dbus.ObjectPath]InterfacesAndProperties),
		driveInfo:    s.Props,
	}
}

func (dm *driveMap) addInterface(s *Event) (bool, error) {
	objectPathString := string(s.Path)
	var blockDevice bool

	if strings.HasPrefix(objectPathString, path.Join(dbusObject, "drives")) {
		if _, ok := (*dm)[s.Path]; ok {
			log.Println("WARNING: replacing", s.Path, "with new drive event")
		}
		(*dm)[s.Path] = newDrive(s)
	} else if strings.HasPrefix(objectPathString, path.Join(dbusObject, "block_devices")) {
		driveObjectPath, err := s.getDrive()
		if err != nil {
			return blockDevice, err
		}
		if _, ok := (*dm)[driveObjectPath]; !ok {
			return blockDevice, errors.New("drive holding block device is not mapped")
		}
		(*dm)[driveObjectPath].blockDevices[s.Path] = s.Props
		blockDevice = true
	} else {
		// we don't care about other object paths
	}

	return blockDevice, nil
}
