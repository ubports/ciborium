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
	"log"
	"time"

	"launchpad.net/ciborium/notifications"
	"launchpad.net/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
)

type message struct{ Summary, Body string }

var (
	msgStorageSucces message = message{
		Summary: "Storage device detected",
		Body:    "This device will be scanned for new content",
	}

	msgStorageFail message = message{
		Summary: "Failed to add storage device",
		Body:    "Make sure the storage device is correctly formated",
	}
)

var supportedFS []string = []string{"vfat"}

func main() {
	var (
		systemBus, sessionBus *dbus.Connection
		err                   error
	)

	if systemBus, err = dbus.Connect(dbus.SystemBus); err != nil {
		log.Fatal("Connection error: ", err)
	}
	log.Print("Using system bus on ", systemBus.UniqueName)

	if sessionBus, err = dbus.Connect(dbus.SessionBus); err != nil {
		log.Fatal("Connection error: ", err)
	}
	log.Print("Using session bus on ", sessionBus.UniqueName)

	udisks2, err := udisks2.NewStorageWatcher(systemBus, supportedFS...)
	if err != nil {
		log.Fatal(err)
	}

	timeout := time.Second * 4
	n := notifications.NewNotificationHandler(sessionBus, "ciborium", "system-settings", timeout)

	go func() {
		for a := range udisks2.DriveAdded {
			if mountpoint, err := a.Mount(systemBus); err != nil {
				if err := n.SimpleNotify(msgStorageFail.Summary, msgStorageFail.Body); err != nil {
					log.Println(err)
				}
			} else {
				fmt.Println("Mounted", mountpoint)
				if err := n.SimpleNotify(msgStorageSucces.Summary, msgStorageSucces.Body); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	done := make(chan bool)
	<-done
}
