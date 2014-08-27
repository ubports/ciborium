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
	"encoding/json"
	"path"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName       = "com.ubuntu.Postal"
	dbusInterface  = "com.ubuntu.Postal"
	dbusPathPart   = "/com/ubuntu/Postal/"
	dbusPostMethod = "Post"
)

type VariantMap map[string]dbus.Variant

type notificationHandler struct {
	dbusObject  *dbus.ObjectProxy
	application string
}

func NewNotificationHandler(conn *dbus.Connection, application string) *notificationHandler {
	return &notificationHandler{
		dbusObject:  conn.Object(dbusName, dbus.ObjectPath(path.Join(dbusPathPart, application))),
		application: application,
	}
}

func (n *notificationHandler) Send(m *PushMessage) error {
	var pushMessage string
	if out, err := json.Marshal(m); err == nil {
		pushMessage = string(out)
	} else {
		return err
	}
	if _, err := n.dbusObject.Call(dbusInterface, dbusPostMethod, n.application, pushMessage); err != nil {
		return err
	}
	return nil
}

// NewStandardPushMessage creates a base Notification with common
// components (members) setup.
func (n *notificationHandler) NewStandardPushMessage(summary, body, icon string) *PushMessage {
	pm := &PushMessage{
		Notification: Notification{
			Card: &Card{
				Summary: summary,
				Body:    body,
				Actions: []string{"application:///" + n.application + ".desktop"},
				Icon:    icon,
				Popup:   true,
				Persist: true,
			},
		},
	}
	return pm
}

// PushMessage represents a data structure to be sent over to the
// Post Office. It consists of a Notification and a Message.
type PushMessage struct {
	// Notification (optional) describes the user-facing notifications
	// triggered by this push message.
	Notification Notification `json:"notification,omitempty"`
}

// Notification (optional) describes the user-facing notifications
// triggered by this push message.
type Notification struct {
	Card *Card `json:"card,omitempty"`
}

// Card is part of a notification and represents the user visible hints for
// a specific notification.
type Card struct {
	// Summary is a required title. The card will not be presented if this is missing.
	Summary string `json:"summary"`
	// Body is the longer text.
	Body string `json:"body,omitempty"`
	// Whether to show a bubble. Users can disable this, and can easily miss
	// them, so don’t rely on it exclusively.
	Popup bool `json:"popup,omitempty"`
	// Actions provides actions for the bubble's snap decissions.
	Actions []string `json:"actions,omitempty"`
	// Icon is a path to an icon to display with the notification bubble.
	Icon string `json:"icon,omitempty"`
	// Whether to show in notification centre.
	Persist bool `json:"persist,omitempty"`
}
