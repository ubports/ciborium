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

	"gopkg.in/qml.v0"
	"launchpad.net/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
)

const (
	qmlPath = "share/qml"
	mainQml = "main.qml"
)

type driveControl struct {
	udisks         *udisks2.UDisks2
	ExternalDrives []udisks2.Drive
}

var mainQmlPath = filepath.Join(qmlPath, mainQml)
var supportedFS []string = []string{"vfat"}

func init() {
	os.Setenv("APP_ID", "ciborium")
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		p := filepath.Join(goPath, "src", goSource(), mainQml)
		if _, err := os.Stat(p); err == nil {
			mainQmlPath = p
		}
	}
}

func goSource() string {
	return filepath.Join("launchpad.net", "ciborium")
}

func main() {
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

	//ctrl.Root = window.Root()

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

func (ctrl *driveControl) Drives() {
	go func() {
		ctrl.ExternalDrives = ctrl.udisks.ExternalDrives()
		qml.Changed(ctrl, &ctrl.ExternalDrives)
		fmt.Println(ctrl.ExternalDrives)
	}()
}
