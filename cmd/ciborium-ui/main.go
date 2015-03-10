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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"log"

	"launchpad.net/ciborium/qml.v0"
	"launchpad.net/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
	"launchpad.net/go-xdg/v0"
)

type driveControl struct {
	udisks         *udisks2.UDisks2
	ExternalDrives []udisks2.Drive
	Len            int
	Formatting     bool
	FormatError    bool
	Unmounting     bool
	UnmountError   bool
	DevicePresent  bool
}

type DriveList struct {
	Len            int
	ExternalDrives []udisks2.Drive
}

var mainQmlPath = filepath.Join("ciborium", "qml", "main.qml")
var supportedFS []string = []string{"vfat"}

func init() {
	os.Setenv("APP_ID", "ciborium")
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		p := filepath.Join(goPath, "src", goSource(), "share", mainQmlPath)
		fmt.Println(p)
		if _, err := os.Stat(p); err == nil {
			mainQmlPath = p
		}
	} else {
		p, err := xdg.Data.Find(mainQmlPath)
		if err != nil {
			log.Fatal("Unable to find main qml:", err)
		}
		mainQmlPath = p
	}
}

func goSource() string {
	return filepath.Join("launchpad.net", "ciborium")
}

func main() {
	// set default logger flags to get more useful info
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	qml.Init(nil)
	engine := qml.NewEngine()
	component, err := engine.LoadFile(mainQmlPath)
	if err != nil {
		log.Fatal(err)
	}

	context := engine.Context()

	driveCtrl, err := newDriveControl()
	if err != nil {
		log.Fatal(err)
	}
	context.SetVar("driveCtrl", driveCtrl)

	window := component.CreateWindow(nil)
	rand.Seed(time.Now().Unix())

	window.Show()
	window.Wait()
}

func newDriveControl() (*driveControl, error) {
	systemBus, err := dbus.Connect(dbus.SystemBus)
	if err != nil {
		return nil, err
	}
	udisks := udisks2.NewStorageWatcher(systemBus, supportedFS...)

	if err := udisks.Init(); err != nil {
		return nil, err
	}

	return &driveControl{udisks: udisks}, nil
}

func (ctrl *driveControl) Watch() {
	c := ctrl.udisks.SubscribeBlockDeviceEvents()
	go func() {
		ctrl.Drives()
		for block := range c {
			if block {
				log.Println("Block device added")
				ctrl.DevicePresent = true
				qml.Changed(ctrl, &ctrl.DevicePresent)
			} else {
				log.Println("Block device removed")
				// TODO: keep track of the removed device
				ctrl.DevicePresent = false
				qml.Changed(ctrl, &ctrl.DevicePresent)
			}
			ctrl.Drives()
		}
	}()
	// deal with the format jobs so that we do show the dialog correctly
	go func() {
		formatDone, formatErrors := ctrl.udisks.SubscribeFormatEvents()
		for {
			select {
			case d := <-formatDone:
				log.Println("Formatting job done", d)
				ctrl.Formatting = false
				qml.Changed(ctrl, &ctrl.Formatting)
				ctrl.FormatError = false
				qml.Changed(ctrl, &ctrl.FormatError)
			case e := <-formatErrors:
				log.Println("Formatting job error", e)
				ctrl.FormatError = true
				qml.Changed(ctrl, &ctrl.FormatError)
			}
		}
	}()

	// deal with mount and unmount events so that the ui is updated accordingly
	go func() {
		mountCompleted, mountErrors := ctrl.udisks.SubscribeMountEvents()
		unmountCompleted, unmountErrors := ctrl.udisks.SubscribeUnmountEvents()
		for {
			select {
			case d := <-mountCompleted:
				log.Println("Mount job done", d)
				// get the drive and state that it was mounted
				for index, drive := range ctrl.ExternalDrives {
					if drive.Path == d.Path {
						// grab the drive, set it to mounted and update the qml
						log.Println("Drive", drive.Path, "set to be mounted.")
						drive.Mounted = true
					}
				}
			case e := <-mountErrors:
				log.Println("Mount job error", e)
			case d := <-unmountCompleted:
				log.Println("Unmount job done", d)
				ctrl.Unmounting = false
				qml.Changed(ctrl, &ctrl.Unmounting)
				for index, drive := range ctrl.ExternalDrives {
					if string(drive.Path) == d {
						// grab the drive, set it to mounted and update the qml
						log.Println("Drive", drive.Path, "set to be unmounted.")
						drive.Mounted = false
					}
				}
			case e := <-unmountErrors:
				log.Println("Unmount job error", e)
				ctrl.UnmountError = true
				qml.Changed(ctrl, &ctrl.UnmountError)
			}
		}
	}()

}

func (ctrl *driveControl) Drives() {
	log.Println("Get present drives.")
	go func() {
		ctrl.ExternalDrives = ctrl.udisks.ExternalDrives()
		ctrl.Len = len(ctrl.ExternalDrives)
		qml.Changed(ctrl, &ctrl.ExternalDrives)
		qml.Changed(ctrl, &ctrl.Len)
	}()
}

func (ctrl *driveControl) DriveModel(index int) string {
	return ctrl.ExternalDrives[index].Model()
}

func (ctrl *driveControl) DriveFormat(index int) {
	ctrl.Formatting = true
	ctrl.FormatError = false
	ctrl.UnmountError = false
	qml.Changed(ctrl, &ctrl.Formatting)

	drive := ctrl.ExternalDrives[index]

	log.Println("Format drive on index", index, "model", drive.Model(), "path", drive.Path)
	ctrl.udisks.Format(&drive)
}

func (ctrl *driveControl) DriveUnmount(index int) {
	log.Println("Unmounting device.")
	drive := ctrl.ExternalDrives[index]
	ctrl.Unmounting = true
	qml.Changed(ctrl, &ctrl.Unmounting)
	ctrl.udisks.Unmount(&drive)
}
