/*
 * Copyright 2015 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@cannical.com
 * Manuel de la Pena: manuel.delapena@canonical.com
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * ciborium is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package udisks2

import (
	"log"
	"reflect"

	"launchpad.net/go-dbus/v1"
)

const (
	formatErase           = "format-erase"
	formateMkfs           = "format-mkfs"
	unmountFs             = "filesystem-unmount"
	mountFs               = "filesystem-mount"
	mountPointsProperty   = "MountPoints"
	uuidProperty          = "UUID"
	tableProperty         = "Table"
	partitionableProperty = "HintPartitionable"
	operationProperty     = "Operation"
	objectsProperty       = "Objects"
)

type VariantMap map[string]dbus.Variant
type InterfacesAndProperties map[string]VariantMap

func (i InterfacesAndProperties) isMounted() bool {
	propFS, ok := i[dbusFilesystemInterface]
	if !ok {
		return false
	}
	mountpointsVariant, ok := propFS[mountPointsProperty]
	if !ok {
		return false
	}
	if reflect.TypeOf(mountpointsVariant.Value).Kind() != reflect.Slice {
		return false
	}
	mountpoints := reflect.ValueOf(mountpointsVariant.Value).Len()
	log.Println("Mount points found:", mountpoints)

	return mountpoints > 0
}

func (i InterfacesAndProperties) hasPartition() bool {
	prop, ok := i[dbusPartitionInterface]
	if !ok {
		return false
	}
	// check if a couple of properties exist
	if _, ok := prop[uuidProperty]; !ok {
		return false
	}
	if _, ok := prop[tableProperty]; !ok {
		return false
	}
	return true
}

func (i InterfacesAndProperties) isPartitionable() bool {
	prop, ok := i[dbusBlockInterface]
	if !ok {
		return false
	}
	partitionableHintVariant, ok := prop[partitionableProperty]
	if !ok {
		return false
	}
	if reflect.TypeOf(partitionableHintVariant.Value).Kind() != reflect.Bool {
		return false
	}
	return reflect.ValueOf(partitionableHintVariant.Value).Bool()
}

func (i InterfacesAndProperties) jobOperation() string {
	prop, ok := i[dbusJobInterface]
	if !ok {
		return ""
	}
	operationVariant, ok := prop[operationProperty]
	if !ok {
		return ""
	}
	if reflect.TypeOf(operationVariant.Value).Kind() != reflect.String {
		return ""
	}
	return reflect.ValueOf(operationVariant.Value).String()
}

func (i InterfacesAndProperties) isEraseFormatJob() bool {
	return i.jobOperation() == formatErase

}

func (i InterfacesAndProperties) isMkfsFormatJob() bool {
	return i.jobOperation() == formateMkfs
}

func (i InterfacesAndProperties) isUnmountJob() bool {
	return i.jobOperation() == unmountFs
}

func (i InterfacesAndProperties) isMountJob() bool {
	return i.jobOperation() == mountFs
}

func (i InterfacesAndProperties) getFormattedPaths() []string {
	var objectPaths []string
	prop, ok := i[dbusJobInterface]
	if !ok {
		return objectPaths
	}
	operationVariant, ok := prop[operationProperty]
	if !ok {
		return objectPaths
	}

	operationStr := reflect.ValueOf(operationVariant.Value).String()
	if operationStr == formateMkfs || operationStr == unmountFs || operationStr == mountFs {
		objs, ok := prop[objectsProperty]
		if ok {
			objsVal := reflect.ValueOf(objs.Value)
			length := objsVal.Len()
			objectPaths = make([]string, length, length)
			for i := 0; i < length; i++ {
				objectPaths[i] = objsVal.Index(i).Elem().String()
			}
			return objectPaths
		}
	}

	return objectPaths
}

func (i InterfacesAndProperties) isFilesystem() bool {
	_, ok := i[dbusFilesystemInterface]
	return ok
}
