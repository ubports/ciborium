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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ubports/ciborium/gettext"
	"github.com/ubports/ciborium/notifications"
	"github.com/ubports/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
)

type message struct{ Summary, Body string }
type notifyFreeFunc func(mountpoint) error

type mountpoint string

func (m mountpoint) external() bool {
	return strings.HasPrefix(string(m), "/media")
}

type mountwatch struct {
	lock        sync.Mutex
	mountpoints map[mountpoint]bool
}

func (m *mountwatch) set(path mountpoint, state bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.mountpoints[path] = state
}

func (m *mountwatch) getMountpoints() []mountpoint {
	m.lock.Lock()
	defer m.lock.Unlock()

	mapLen := len(m.mountpoints)
	mountpoints := make([]mountpoint, 0, mapLen)
	for p := range m.mountpoints {
		mountpoints = append(mountpoints, p)
	}
	return mountpoints
}

func (m *mountwatch) warn(path mountpoint) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.mountpoints[path]
}

func (m *mountwatch) remove(path mountpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.mountpoints, path)
}

func newMountwatch() *mountwatch {
	return &mountwatch{
		mountpoints: make(map[mountpoint]bool),
	}
}

const (
	sdCardIcon                = "/usr/share/ciborium/icons/ciborium.svg"
	errorIcon                 = "error"
	homeMountpoint mountpoint = "/home"
	freeThreshold             = 5
)

var (
	mw          *mountwatch
	supportedFS []string
)

func init() {
	mw = newMountwatch()
	mw.set(homeMountpoint, true)
	supportedFS = []string{"vfat"}
}

func main() {
	// set default logger flags to get more useful info
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize i18n
	gettext.SetLocale(gettext.LC_ALL, "")
	gettext.Textdomain("ciborium")
	gettext.BindTextdomain("ciborium", "/usr/share/locale")

	var (
		msgStorageSuccess message = message{
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

	notificationHandler := notifications.NewLegacyHandler(sessionBus, "ciborium")
	notifyFree := buildFreeNotify(notificationHandler)

	blockAdded, blockError := udisks2.SubscribeAddEvents()
	formatCompleted, formatErrors := udisks2.SubscribeFormatEvents()
	unmountCompleted, unmountErrors := udisks2.SubscribeUnmountEvents()
	mountCompleted, mountErrors := udisks2.SubscribeMountEvents()
	mountRemoved := udisks2.SubscribeRemoveEvents()

	// create a routine per couple of channels, the select algorithm will make use
	// ignore some events if more than one channels is being written to the algorithm
	// will pick one at random but we want to make sure that we always react, the pairs
	// are safe since the deal with complementary events

	// block additions
	go func() {
		log.Println("Listening for addition and removal events.")
		for {
			var n *notifications.PushMessage
			select {
			case a := <-blockAdded:
				udisks2.Mount(a)
			case e := <-blockError:
				log.Println("Issues in block for added drive:", e)
				n = notificationHandler.NewStandardPushMessage(
					msgStorageFail.Summary,
					msgStorageFail.Body,
					errorIcon,
				)
			case m := <-mountRemoved:
				log.Println("Path removed", m)
				n = notificationHandler.NewStandardPushMessage(
					msgStorageRemoved.Summary,
					msgStorageRemoved.Body,
					sdCardIcon,
				)
				mw.remove(mountpoint(m))
			case <-time.After(time.Minute):
				for _, m := range mw.getMountpoints() {
					err = notifyFree(m)
					if err != nil {
						log.Print("Error while querying free space for ", m, ": ", err)
					}
				}
			}
			if n != nil {
				if err := notificationHandler.Send(n); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	// mount operations
	go func() {
		log.Println("Listening for mount and unmount events.")
		for {
			var n *notifications.PushMessage
			select {
			case m := <-mountCompleted:
				log.Println("Mounted", m)
				n = notificationHandler.NewStandardPushMessage(
					msgStorageSuccess.Summary,
					msgStorageSuccess.Body,
					sdCardIcon,
				)

				if err := createStandardHomeDirs(m.Mountpoint); err != nil {
					log.Println("Failed to create standard dir layout:", err)
				}

				mw.set(mountpoint(m.Mountpoint), true)
			case e := <-mountErrors:
				log.Println("Error while mounting device", e)

				n = notificationHandler.NewStandardPushMessage(
					msgStorageFail.Summary,
					msgStorageFail.Body,
					errorIcon,
				)
			case m := <-unmountCompleted:
				log.Println("Path removed", m)
				n = notificationHandler.NewStandardPushMessage(
					msgStorageRemoved.Summary,
					msgStorageRemoved.Body,
					sdCardIcon,
				)
				mw.remove(mountpoint(m))
			case e := <-unmountErrors:
				log.Println("Error while unmounting device", e)

				n = notificationHandler.NewStandardPushMessage(
					msgStorageFail.Summary,
					msgStorageFail.Body,
					errorIcon,
				)
			}

			if n != nil {
				if err := notificationHandler.Send(n); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	// format operations
	go func() {
		log.Println("Listening for format events.")
		for {
			var n *notifications.PushMessage
			select {
			case f := <-formatCompleted:
				log.Println("Format done. Trying to mount.")
				udisks2.Mount(f)
			case e := <-formatErrors:
				log.Println("There was an error while formatting", e)
				n = notificationHandler.NewStandardPushMessage(
					msgStorageFail.Summary,
					msgStorageFail.Body,
					errorIcon,
				)
			}

			if n != nil {
				if err := notificationHandler.Send(n); err != nil {
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

// createStandardHomeDirs creates directories reflecting a standard home, these
// directories are Documents, Downloads, Music, Pictures and Videos
func createStandardHomeDirs(mountpoint string) error {
	log.Println("createStandardHomeDirs(", mountpoint, ")")
	for _, node := range []string{"Documents", "Downloads", "Music", "Pictures", "Videos"} {
		dir := filepath.Join(mountpoint, node)

		if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 755); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}

// notify only notifies if a notification is actually needed
// depending on freeThreshold and on warningSent's status
func buildFreeNotify(nh *notifications.NotificationHandler) notifyFreeFunc {
	// TRANSLATORS: This is the summary of a notification bubble with a short message warning on
	// low space
	summary := gettext.Gettext("Low on disk space")
	// TRANSLATORS: This is the body of a notification bubble with a short message about content
	// reamining available space, %d is the remaining percentage of space available on internal
	// storage
	bodyInternal := gettext.Gettext("Only %d%% is available on the internal storage device")
	// TRANSLATORS: This is the body of a notification bubble with a short message about content
	// reamining available space, %d is the remaining percentage of space available on a given
	// external storage device
	bodyExternal := gettext.Gettext("Only %d%% is available on the external storage device")

	var body string

	return func(path mountpoint) error {
		if path.external() {
			body = bodyExternal
		} else {
			body = bodyInternal
		}

		availPercentage, err := queryFreePercentage(path)
		if err != nil {
			return err
		}

		if mw.warn(path) && availPercentage <= freeThreshold {
			n := nh.NewStandardPushMessage(
				summary,
				fmt.Sprintf(body, availPercentage),
				errorIcon,
			)
			log.Println("Warning for", path, "available percentage", availPercentage)
			if err := nh.Send(n); err != nil {
				return err
			}
			mw.set(path, false)
		}

		if availPercentage > freeThreshold {
			mw.set(path, true)
		}
		return nil
	}
}

func queryFreePercentage(path mountpoint) (uint64, error) {
	s := syscall.Statfs_t{}
	if err := syscall.Statfs(string(path), &s); err != nil {
		return 0, err
	}
	if s.Blocks == 0 {
		return 0, errors.New("statfs call returned 0 blocks available")
	}
	return uint64(s.Bavail) * 100 / uint64(s.Blocks), nil
}
