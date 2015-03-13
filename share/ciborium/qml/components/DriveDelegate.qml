import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

UbuntuShape {
    property var onFormatClicked
    property var onSafeRemovalClicked
    property var drive: nil
    property int driveIndex 
    property bool mounted: (drive)?drive.Mounted:false

    height: childrenRect.height + (3 *units.gu(1))
    color: driveIndex % 2 === 0 ? "white" : "#DECAE3"

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
        onClicked: onFormatClicked(formatButton) 

	anchors {
	    top: driveIcon.bottom
	    topMargin: units.gu(1)
	    left: parent.left
	    leftMargin: units.gu(1)

	}
    }

    Button {
        id: removalButton
	enabled: drive.mounted
        text: i18n.tr("Safely Remove")
        onClicked: onSafeRemovalClicked(removalButton)

	anchors {
	    top: driveIcon.bottom
	    topMargin: units.gu(1)
	    left: formatButton.right
	    leftMargin: units.gu(1)
	}
    }

    onMountedChanged: {
        if (mounted) {
            removalButton.enabled = true;
	} else {
            removalButton.enabled = false;
	}
    }

}
