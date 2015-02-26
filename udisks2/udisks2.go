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
	"sync"

	"log"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName                    = "org.freedesktop.UDisks2"
	dbusObject                  = "/org/freedesktop/UDisks2"
	dbusObjectManagerInterface  = "org.freedesktop.DBus.ObjectManager"
	dbusBlockInterface          = "org.freedesktop.UDisks2.Block"
	dbusDriveInterface          = "org.freedesktop.UDisks2.Drive"
	dbusFilesystemInterface     = "org.freedesktop.UDisks2.Filesystem"
	dbusPartitionInterface      = "org.freedesktop.UDisks2.Partition"
	dbusPartitionTableInterface = "org.freedesktop.UDisks2.PartitionTable"
	dbusJobInterface            = "org.freedesktop.UDisks2.Job"
	dbusAddedSignal             = "InterfacesAdded"
	dbusRemovedSignal           = "InterfacesRemoved"
)

var ErrUnhandledFileSystem = errors.New("unhandled filesystem")

type Drive struct {
	path         dbus.ObjectPath
	blockDevices map[dbus.ObjectPath]InterfacesAndProperties
	driveInfo    InterfacesAndProperties
}

type driveMap map[dbus.ObjectPath]*Drive

type mountpointMap map[dbus.ObjectPath]string

type UDisks2 struct {
	conn          *dbus.Connection
	validFS       sort.StringSlice
	blockAdded    chan *Event
	driveAdded    *dbus.SignalWatch
	mountRemoved  chan string
	blockError    chan error
	driveRemoved  *dbus.SignalWatch
	blockDevice   chan bool
	drives        driveMap
	mountpoints   mountpointMap
	mapLock       sync.Mutex
	startLock     sync.Mutex
	dispatcher    *dispatcher
	jobs          *jobManager
	pendingMounts []string
}

func NewStorageWatcher(conn *dbus.Connection, filesystems ...string) (u *UDisks2) {
	u = &UDisks2{
		conn:          conn,
		validFS:       sort.StringSlice(filesystems),
		drives:        make(driveMap),
		mountpoints:   make(mountpointMap),
		pendingMounts: make([]string, 0, 0),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func (u *UDisks2) SubscribeAddEvents() (<-chan *Event, <-chan error) {
	u.blockAdded = make(chan *Event)
	u.blockError = make(chan error)
	return u.blockAdded, u.blockError
}

func (u *UDisks2) SubscribeRemoveEvents() <-chan string {
	u.mountRemoved = make(chan string)
	return u.mountRemoved
}

func (u *UDisks2) SubscribeBlockDeviceEvents() <-chan bool {
	u.blockDevice = make(chan bool)
	return u.blockDevice
}

func (u *UDisks2) Mount(s *Event) (mountpoint string, err error) {
	obj := u.conn.Object(dbusName, s.Path)
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
	log.Println("Mounth path for '", s.Path, "' set to be", mountpoint)
	return mountpoint, err
}

func (u *UDisks2) Unmount(d *Drive) error {
	for blockPath, block := range d.blockDevices {
		if block.isMounted() {
			if err := u.umount(blockPath); err != nil {
				log.Println("Issues while unmounting", blockPath, ":", err)
				return err
			}
			if _, ok := u.mountpoints[blockPath]; ok {
				delete(u.mountpoints, blockPath)
			}
		} else {
			log.Println(blockPath, "is not mounted")
		}
	}
	return nil
}

func (u *UDisks2) umount(o dbus.ObjectPath) error {
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusFilesystemInterface, "Unmount", options)
	return err
}

func (u *UDisks2) Format(d *Drive) error {
	if err := u.Unmount(d); err != nil {
		log.Println("Error while unmounting:", err)
		return err
	}
	// delete all the partitions
	for blockPath, block := range d.blockDevices {
		if block.hasPartition() {
			if err := u.deletePartition(blockPath); err != nil {
				log.Println("Issues while deleting partition on", blockPath, ":", err)
				return err
			}
			// delete the block from the map as it shouldn't exist anymore
			delete(d.blockDevices, blockPath)
		}
	}

	// format the blocks with PartitionTable
	for blockPath, block := range d.blockDevices {
		if !block.isPartitionable() {
			continue
		}
		if err := u.format(blockPath); err != nil {
			return err
		}
	}

	return nil
}

func (u *UDisks2) format(o dbus.ObjectPath) error {
	log.Println("Formatting", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusBlockInterface, "Format", "vfat", options)
	return err
}

func (u *UDisks2) deletePartition(o dbus.ObjectPath) error {
	log.Println("Calling delete on", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusPartitionInterface, "Delete", options)
	return err
}

func (u *UDisks2) ExternalDrives() []Drive {
	u.startLock.Lock()
	defer u.startLock.Unlock()
	var drives []Drive
	for _, d := range u.drives {
		if !d.hasSystemBlockDevices() && len(d.blockDevices) != 0 {
			drives = append(drives, *d)
		}
	}
	return drives
}

func (u *UDisks2) Init() (err error) {
	d, err := newDispatcher(u.conn)
	if err == nil {
		u.dispatcher = d
		u.jobs = newJobManager(d)
		go func() {
			for {
				select {
				case e := <-u.dispatcher.Additions:
					if err := u.processAddEvent(&e); err != nil {
						log.Print("Issues while processing ", e.Path, ": ", err)
					}
				case e := <-u.dispatcher.Removals:
					if err := u.processRemoveEvent(e.Path, e.Interfaces); err != nil {
						log.Println("Issues while processing remove event:", err)
					}
				case j := <-u.jobs.FormatEraseJobs:
					if j.WasCompleted {
						log.Print("Erase job completed.")
					} else {
						log.Print("Erase job started.")
					}
				case j := <-u.jobs.FormatMkfsJobs:
					if j.WasCompleted {
						log.Println("Format job done for", j.Event.Path)
						u.pendingMounts = append(u.pendingMounts, j.Paths...)
						sort.Strings(u.pendingMounts)
					} else {
						log.Print("Format job started.")
					}
				}
			}
		}()
		d.Init()
		u.emitExistingDevices()
		return nil
	}
	return err
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
	u.startLock.Lock()
	defer u.startLock.Unlock()
	obj := u.conn.Object(dbusName, dbusObject)
	reply, err := obj.Call(dbusObjectManagerInterface, "GetManagedObjects")
	if err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}

	allDevices := make(map[dbus.ObjectPath]InterfacesAndProperties)
	if err := reply.Args(&allDevices); err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}

	var blocks, drives []*Event
	// separate drives from blocks to avoid aliasing
	for objectPath, props := range allDevices {
		s := &Event{objectPath, props, make([]string, 0, 0)}
		switch objectPathType(objectPath) {
		case deviceTypeDrive:
			drives = append(drives, s)
		case deviceTypeBlock:
			blocks = append(blocks, s)
		}
	}

	for i := range drives {
		if err := u.processAddEvent(drives[i]); err != nil {
			log.Println("Error while processing events:", err)
		}
	}

	for i := range blocks {
		if err := u.processAddEvent(blocks[i]); err != nil {
			log.Println("Error while processing events:", err)
		}
	}
}

func (u *UDisks2) processAddEvent(s *Event) error {
	log.Println("processAddEvents(", s.Path, ")")
	u.mapLock.Lock()
	defer u.mapLock.Unlock()

	pos := sort.SearchStrings(u.pendingMounts, string(s.Path))
	if pos != len(u.pendingMounts) && s.Props.isFilesystem() {
		log.Println("Mount path", s.Path)
		_, err := u.Mount(s)
		u.pendingMounts = append(u.pendingMounts[:pos], u.pendingMounts[pos+1:]...)
		if err != nil {
			log.Println("Error mounting new path")
			u.blockError <- err
			return err
		}
	}
	if isBlockDevice, err := u.drives.addInterface(s); err != nil {
		return err
	} else if isBlockDevice {
		if u.blockAdded != nil && u.blockError != nil {
			if ok, err := u.desiredMountableEvent(s); err != nil {
				u.blockError <- err
			} else if ok {
				u.blockAdded <- s
			}
		}
		if u.blockDevice != nil {
			u.blockDevice <- true
		}
	}

	return nil
}

func (u *UDisks2) processRemoveEvent(objectPath dbus.ObjectPath, interfaces Interfaces) error {
	log.Println("Remove event for", objectPath)
	mountpoint, mounted := u.mountpoints[objectPath]
	if mounted {
		log.Println("Removing mountpoint", mountpoint)
		delete(u.mountpoints, objectPath)
		if u.mountRemoved != nil && interfaces.desiredUnmountEvent() {
			u.mountRemoved <- mountpoint
		} else {
			return errors.New("mounted but does not remove filesystem interface")
		}
	}
	u.mapLock.Lock()
	log.Println("Removing device", objectPath)
	if strings.HasPrefix(string(objectPath), path.Join(dbusObject, "drives")) {
		delete(u.drives, objectPath)
	} else {
		// TODO: remove filesystem interface from map
	}
	u.mapLock.Unlock()
	if u.blockDevice != nil {
		u.blockDevice <- false
	}
	return nil
}

func cleanDriveWatch(u *UDisks2) {
	log.Print("Cancelling Interfaces signal watch")
	u.driveAdded.Cancel()
	u.driveRemoved.Cancel()
}

func (iface Interfaces) desiredUnmountEvent() bool {
	for i := range iface {
		fmt.Println(iface[i])
		if iface[i] == dbusFilesystemInterface {
			return true
		}
	}
	return false
}

func (u *UDisks2) desiredMountableEvent(s *Event) (bool, error) {
	// No file system interface means we can't mount it even if we wanted to
	_, ok := s.Props[dbusFilesystemInterface]
	if !ok {
		return false, nil
	}

	drivePath, err := s.getDrive()
	if err != nil {
		//log.Println("Issues while getting drive:", err)
		return false, nil
	}

	drive := u.drives[drivePath]
	if ok := drive.hasSystemBlockDevices(); ok {
		//log.Println(drivePath, "which contains", s.Path, "has HintSystem set")
		return false, nil
	}

	driveProps, ok := drive.driveInfo[dbusDriveInterface]
	if !ok {
		//log.Println(drivePath, "doesn't hold a Drive interface")
		return false, nil
	}
	if mediaRemovableVariant, ok := driveProps["MediaRemovable"]; !ok {
		//log.Println(drivePath, "which holds", s.Path, "doesn't have MediaRemovable")
		return false, nil
	} else {
		mediaRemovable := reflect.ValueOf(mediaRemovableVariant.Value).Bool()
		if !mediaRemovable {
			//log.Println(drivePath, "which holds", s.Path, "is not MediaRemovable")
			return false, nil
		}
	}

	if s.Props.isMounted() {
		return false, nil
	}

	propBlock, ok := s.Props[dbusBlockInterface]
	if !ok {
		return false, nil
	}
	id, ok := propBlock["IdType"]
	if !ok {
		log.Println(s.Path, "doesn't hold IdType")
		return false, nil
	}

	fs := reflect.ValueOf(id.Value).String()
	if fs == "" {
		return false, nil
	}

	i := u.validFS.Search(fs)
	if i >= u.validFS.Len() || u.validFS[i] != fs {
		log.Println(fs, "not in:", u.validFS, "for", s.Path)
		return false, ErrUnhandledFileSystem
	}

	return true, nil
}

func (d *Drive) hasSystemBlockDevices() bool {
	for _, blockDevice := range d.blockDevices {
		if propBlock, ok := blockDevice[dbusBlockInterface]; ok {
			if systemHintVariant, ok := propBlock["HintSystem"]; ok {
				return reflect.ValueOf(systemHintVariant.Value).Bool()
			}
		}
	}
	return false
}

func (d *Drive) Model() string {
	propDrive, ok := d.driveInfo[dbusDriveInterface]
	if !ok {
		return ""
	}
	modelVariant, ok := propDrive["Model"]
	if !ok {
		return ""
	}
	return reflect.ValueOf(modelVariant.Value).String()
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

func newDrive(s *Event) *Drive {
	return &Drive{
		path:         s.Path,
		blockDevices: make(map[dbus.ObjectPath]InterfacesAndProperties),
		driveInfo:    s.Props,
	}
}

const (
	deviceTypeBlock = iota
	deviceTypeDrive
	deviceTypeUnhandled
)

type dbusObjectPathType uint

func objectPathType(objectPath dbus.ObjectPath) dbusObjectPathType {
	objectPathString := string(objectPath)
	if strings.HasPrefix(objectPathString, path.Join(dbusObject, "drives")) {
		return deviceTypeDrive
	} else if strings.HasPrefix(objectPathString, path.Join(dbusObject, "block_devices")) {
		return deviceTypeBlock
	} else {
		return deviceTypeUnhandled
	}
}

func (dm *driveMap) addInterface(s *Event) (bool, error) {
	var blockDevice bool

	switch objectPathType(s.Path) {
	case deviceTypeDrive:
		if _, ok := (*dm)[s.Path]; ok {
			log.Println("WARNING: replacing", s.Path, "with new drive event")
		}
		(*dm)[s.Path] = newDrive(s)
	case deviceTypeBlock:
		driveObjectPath, err := s.getDrive()
		if err != nil {
			return blockDevice, err
		}
		if _, ok := (*dm)[driveObjectPath]; !ok {
			return blockDevice, errors.New("drive holding block device is not mapped")
		}
		(*dm)[driveObjectPath].blockDevices[s.Path] = s.Props
		blockDevice = true
	default:
		// we don't care about other object paths
		log.Println("Unhandled object path", s.Path)
	}

	return blockDevice, nil
}
