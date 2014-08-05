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

// Notifications lives on a well-knwon bus.Address

package notifications

import (
	"time"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName         = "org.freedesktop.Notifications"
	dbusInterface    = "org.freedesktop.Notifications"
	dbusPath         = "/org/freedesktop/Notifications"
	dbusNotifyMethod = "Notify"
)

type VariantMap map[string]dbus.Variant

type Notification struct {
	dbusObject    *dbus.ObjectProxy
	appName, icon string
	timeout       int32
}

func NewNotificationHandler(conn *dbus.Connection, appName, icon string, timeout time.Duration) *Notification {
	return &Notification{
		dbusObject: conn.Object(dbusName, dbusPath),
		appName:    appName,
		icon:       icon,
		timeout:    int32(timeout.Seconds()) * 1000,
	}
}

func (n *Notification) SimpleNotify(summary, body string) error {
	return n.Notify(summary, body, nil, nil)
}

func (n *Notification) Notify(summary, body string, actions []string, hints VariantMap) error {
	var reuseId uint32
	if _, err := n.dbusObject.Call(dbusInterface, dbusNotifyMethod, n.appName, reuseId, n.icon, summary, body, actions, hints, n.timeout); err != nil {
		return err
	}
	return nil
}
