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
	dbusAddedSignal             = "InterfacesAdded"
	dbusRemovedSignal           = "InterfacesRemoved"
)

var ErrUnhandledFileSystem = errors.New("unhandled filesystem")

type VariantMap map[string]dbus.Variant
type InterfacesAndProperties map[string]VariantMap
type Interfaces []string

type Drive struct {
	path         dbus.ObjectPath
	blockDevices map[dbus.ObjectPath]InterfacesAndProperties
	driveInfo    InterfacesAndProperties
}

type driveMap map[dbus.ObjectPath]*Drive

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
	BlockError   chan error
	driveRemoved *dbus.SignalWatch
	blockDevice  chan bool
	drives       driveMap
	mountpoints  mountpointMap
	mapLock      sync.Mutex
	startLock    sync.Mutex
}

func NewStorageWatcher(conn *dbus.Connection, filesystems ...string) (u *UDisks2) {
	u = &UDisks2{
		conn:         conn,
		validFS:      sort.StringSlice(filesystems),
		DriveAdded:   make(chan *Event),
		DriveRemoved: make(chan dbus.ObjectPath),
		BlockError:   make(chan error),
		drives:       make(driveMap),
		mountpoints:  make(mountpointMap),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func (u *UDisks2) SubscribeBlockDeviceEvents() chan bool {
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
	return mountpoint, err
}

func (u *UDisks2) Unmount(d *Drive) error {
	for blockPath, block := range d.blockDevices {
		if block.isMounted() {
			if err := u.umount(blockPath); err != nil {
				log.Println("Issues while unmounting", blockPath, ":", err)
				continue
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
	if err != nil {
		return err
	}
	return nil
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
	if err != nil {
		return err
	}
	return nil
}

func (u *UDisks2) deletePartition(o dbus.ObjectPath) error {
	log.Println("Calling delete on", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusPartitionInterface, "Delete", options)
	if err != nil {
		return err
	}
	return nil
}

func (u *UDisks2) ExternalDrives() []Drive {
	u.startLock.Lock()
	defer u.startLock.Unlock()
	var drives []Drive
	for _, d := range u.drives {
		if d.hasSystemBlockDevices() {
			continue
		} else if len(d.blockDevices) == 0 {
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
		s := &Event{objectPath, props}
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
	u.mapLock.Lock()
	defer u.mapLock.Unlock()
	if isBlockDevice, err := u.drives.addInterface(s); err != nil {
		return err
	} else if isBlockDevice {
		if ok, err := u.desiredMountableEvent(s); err != nil {
			u.BlockError <- err
		} else if ok {
			u.DriveAdded <- s
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
		if interfaces.desiredUnmountEvent() {
			u.DriveRemoved <- objectPath
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
	i := u.validFS.Search(fs)
	if i >= u.validFS.Len() || u.validFS[i] != fs {
		log.Println(fs, "not in:", u.validFS, "for", s.Path)
		return false, ErrUnhandledFileSystem
	}

	return true, nil
}

func (d *Drive) hasSystemBlockDevices() bool {
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

func (i InterfacesAndProperties) isMounted() bool {
	propFS, ok := i[dbusFilesystemInterface]
	if !ok {
		return false
	}
	mountpointsVariant, ok := propFS["MountPoints"]
	if !ok {
		return false
	}
	if reflect.TypeOf(mountpointsVariant.Value).Kind() != reflect.Slice {
		return false
	}
	if mountpoints := reflect.ValueOf(mountpointsVariant.Value).Len(); mountpoints > 0 {
		return true
	}
	return false
}

func (i InterfacesAndProperties) hasPartition() bool {
	prop, ok := i[dbusPartitionInterface]
	if !ok {
		return false
	}
	// check if a couple of properties exist
	if _, ok := prop["UUID"]; !ok {
		return false
	}
	if _, ok := prop["Table"]; !ok {
		return false
	}
	return true
}

func (i InterfacesAndProperties) isPartitionable() bool {
	prop, ok := i[dbusBlockInterface]
	if !ok {
		return false
	}
	partitionableHintVariant, ok := prop["HintPartitionable"]
	if !ok {
		return false
	}
	if reflect.TypeOf(partitionableHintVariant.Value).Kind() != reflect.Bool {
		return false
	}
	return reflect.ValueOf(partitionableHintVariant.Value).Bool()
}
