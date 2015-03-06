import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Item {
    property int driveIndex

    height: childrenRect.height
    width: childrenRect.width

    Component {
        id: dialogRemoved

        Dialog {
            id: dialogueRemoved
            property bool isError: driveCtrl.unmountError


            title: i18n.tr("Safe to remove")
            text: i18n.tr("You can now safely remove the device")

            Button {
                text: i18n.tr("Ok")
                color: UbuntuColors.orange
                onClicked: {
                    PopupUtils.close(dialogueRemoved)
                }
            }

            onIsErrorChanged: {
	    	if (isError) {
			dialogueRemoved.title = i18n.tr("Unmount Error");
			dialogueRemoved.text = i18n.tr("The device could not be unmounted because is busy");
		} else {
			dialogueRemoved.title = i18n.tr("Safe to remove");
			dialogueRemoved.text = i18n.tr("You can now safely remove the device");
		}
	    }
        }
    }

    Component {
        id: dialogConfirmRemove

        Dialog {
            id: dialogueConfirmRemove

            title: i18n.tr("Confirm remove")
            text: i18n.tr("Files on the device can't be accessed after removing")

            Button {
                text: i18n.tr("Cancel")
                onClicked: PopupUtils.close(dialogueConfirmRemove)
            }
            Button {
                text: i18n.tr("Continue")
                color: UbuntuColors.orange
                onClicked: {
                    driveCtrl.driveUnmount(index)
                    PopupUtils.close(dialogueConfirmRemove)
                    saferemoval.enabled= false
                    PopupUtils.open(dialogRemoved)
                }
            }
        }
    }

    Button {
        anchors.centerIn: parent
        id: saferemoval
        text: i18n.tr("Safely Remove")
        onClicked: {
            PopupUtils.open(dialogConfirmRemove)
        }
    }
}
