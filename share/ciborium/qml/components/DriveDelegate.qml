import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

UbuntuShape {
    property int driverIndex
    property var parentWindow
    property var formatDialog
    property var removeDialog

    color: driverIndex % 2 === 0 ? "white" : "#DECAE3"

    Icon {
    	id: driveIcon
        width: 24
        height: 24
        name: "media-memory-sd"
        //source: "file:///usr/share/icons/Humanity/devices/48/media-memory-sd.svg"

	anchors {
	    top: parent.top
	    topMargin: units.gu(2)
	    left: parent.left
	    leftMargin: units.gu(2)
	}
    }

    Label {
    	id: driveLabel
        text: driveCtrl.driveModel(index)

	anchors {
            top: parent.top
	    topMargin: units.gu(2)
	    left: driveIcon.right
	    leftMargin: units.gu(2)
	    right: parent.right
	    rightMargin: units.gu(2)
	    bottom:  driveIcon.bottom
	}
    }

    Button {
        id: formatButton
        text: i18n.tr("Format")
        onClicked: PopupUtils.open(formatDialog, parentWindow, {"driverIndex": driverIndex})

	anchors {
	    top: driveIcon.bottom
	    topMaring: units.gu(2)
	    left: parent.left
	    leftMargin: units.gu(2)

	}
    }

    Button {
        id: removalButton
        anchors.centerIn: parent
        text: i18n.tr("Safely Remove")
        // TODO: pass index and dialog id
        onClicked: {
            PopupUtils.open(removeDialog, parentWindows, {"driverIndex": driverIndex})
        }

	anchors {
	    top: driveIcon.bottom
	    topMaring: units.gu(2)
	    left: formatButton.left
	    leftMargin: units.gu(2)
	}
    }
}
