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
	"log"
	"time"

	"launchpad.net/ciborium/gettext"
	"launchpad.net/ciborium/notifications"
	"launchpad.net/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
)

type message struct{ Summary, Body string }

var supportedFS []string = []string{"vfat"}

func main() {

	// Initialize i18n
	gettext.SetLocale(gettext.LC_ALL, "")
	gettext.Textdomain("ciborium")
	gettext.BindTextdomain("ciborium", "/usr/share/locale")

	var (
		msgStorageSucces message = message{
			// TRANSLATORS: This is the summary of a notification bubble with a short message of
			// success when addding a storage device.
			Summary: gettext.Gettext("Storage device detected"),
			// TRANSLATORS: This is the body of a notification bubble with a short message about content
			// being scanned when addding a storage device.
			Body: gettext.Gettext("This device will be scanned for new content"),
		}

		msgStorageFail message = message{
			// TRANSLATORS: This is the summary of a notification bubble with a short message of
			// failure when adding a storage device.
			Summary: gettext.Gettext("Failed to add storage device"),
			// TRANSLATORS: This is the body of a notification bubble with a short message with hints
			// with regards to the failure when adding a storage device.
			Body: gettext.Gettext("Make sure the storage device is correctly formated"),
		}

		msgStorageRemoved message = message{
			// TRANSLATORS: This is the summary of a notification bubble with a short message of
			// a storage device being removed
			Summary: gettext.Gettext("Storage device has been removed"),
			// TRANSLATORS: This is the body of a notification bubble with a short message about content
			// from the removed device no longer being available
			Body: gettext.Gettext("Content previously available on this device will no longer be accessible"),
		}
	)

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

	udisks2 := udisks2.NewStorageWatcher(systemBus, supportedFS...)

	timeout := time.Second * 4
	n := notifications.NewNotificationHandler(sessionBus, "ciborium", "system-settings", timeout)

	go func() {
		for {
			select {
			case a := <-udisks2.DriveAdded:
				if mountpoint, err := udisks2.Mount(a); err != nil {
					log.Println("Cannot mount", a.Path, "due to:", err)
					if err := n.SimpleNotify(msgStorageFail.Summary, msgStorageFail.Body); err != nil {
						log.Println(err)
					}
				} else {
					log.Println("Mounted", a.Path, "as", mountpoint)
					if err := n.SimpleNotify(msgStorageSucces.Summary, msgStorageSucces.Body); err != nil {
						log.Println(err)
					}
				}
			case r := <-udisks2.DriveRemoved:
				log.Println("Path removed", r)
				if err := n.SimpleNotify(msgStorageRemoved.Summary, msgStorageRemoved.Body); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	if err := udisks2.Init(); err != nil {
		log.Fatal("Cannot monitor storage devices:", err)
	}

	done := make(chan bool)
	<-done
}
