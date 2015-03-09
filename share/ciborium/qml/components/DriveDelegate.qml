import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Column {
    property int driverIndex
    property var parentWindow
    property var formatDialog
    property var removeDialog

    spacing: units.gu(2)

    Row {
        spacing: units.gu(1)
        height: units.gu(2)
        width: childrenRect.width

        Icon {
            width: 24
            height: 24
            name: "media-memory-sd"
            //source: "file:///usr/share/icons/Humanity/devices/48/media-memory-sd.svg"
        }
        Label {
            width: paintedWidth       
            text: driveCtrl.driveModel(index)
        }
    }

    Row {
        spacing: units.gu(1)
        height: childrenRect.height
        width: childrenRect.width

	Button {
	    anchors.centerIn: parent
            text: i18n.tr("Format")
            onClicked: PopupUtils.open(formatDialog, parentWindow, {"driverIndex": driverIndex})
        }

        Button {
            anchors.centerIn: parent
            text: i18n.tr("Safely Remove")
            // TODO: pass index and dialog id
            onClicked: {
                PopupUtils.open(removeDialog, parentWindows, {"driverIndex": driverIndex})
            }
       }
    }
} 
