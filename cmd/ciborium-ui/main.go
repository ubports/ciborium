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

	"github.com/ubports/ciborium/qml.v1"
	"github.com/ubports/ciborium/udisks2"
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
	return filepath.Join("github.com", "ubports", "ciborium")
}

func main() {
	// set default logger flags to get more useful info
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := qml.Run(run)
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	engine := qml.NewEngine()
	component, err := engine.LoadFile(mainQmlPath)
	if err != nil {
		return err
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
	return nil
}

func newDriveControl() (*driveControl, error) {
	systemBus, err := dbus.Connect(dbus.SystemBus)
	if err != nil {
		return nil, err
	}
	udisks := udisks2.NewStorageWatcher(systemBus, supportedFS...)

	return &driveControl{udisks: udisks}, nil
}

func (ctrl *driveControl) Watch() {
	c := ctrl.udisks.SubscribeBlockDeviceEvents()
	go func() {
		log.Println("Calling Drives from Watch first gorroutine")
		ctrl.Drives()
		for block := range c {
			if block {
				log.Println("Block device added")
			} else {
				log.Println("Block device removed")
			}

			log.Println("Calling Drives from after a block was added or removed")
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
				ctrl.Drives()
			case e := <-mountErrors:
				log.Println("Mount job error", e)
			case d := <-unmountCompleted:
				log.Println("Unmount job done", d)
				ctrl.Unmounting = false
				qml.Changed(ctrl, &ctrl.Unmounting)
			case e := <-unmountErrors:
				log.Println("Unmount job error", e)
				ctrl.UnmountError = true
				qml.Changed(ctrl, &ctrl.UnmountError)
			}
		}
	}()

	ctrl.udisks.Init()
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

func (ctrl *driveControl) DriveAt(index int) *udisks2.Drive {
	return &ctrl.ExternalDrives[index]
}
