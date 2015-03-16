import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Dialog {
    property bool isError: driveCtrl.unmountError
    property int driveCount: driveCtrl.len
    property var onButtonClicked
    property var removeButton
    property var formatButton

    title: i18n.tr("Unmounting")

    Button {
        id: unmountOkButton
        visible: false
        text: i18n.tr("Ok")
        color: UbuntuColors.orange
        onClicked: onButtonClicked()
    }  // Button unmountOkButton

    ActivityIndicator {
        id: unmountActivity
        visible: driveCtrl.unmounting && !isError
        running: driveCtrl.unmounting && !isError
        onRunningChanged: {
            if (!running && !isError) {
                unmountOkButton.visible = true;
                unmountActivity.visible = false;
                text = i18n.tr("You can now safely remove the device");
            }
        } // onRunningChanged
    }  // ActivityIndictor unmountActivity


    onIsErrorChanged: {
        if (isError) {
            title = i18n.tr("Unmount Error");
            text = i18n.tr("The device could not be unmounted because is busy");
	    if (removeButton)
                removeButton.enabled = true
	    if (formatButton)
                formatButton.enabled = false
        } else {
            title = i18n.tr("Safe to remove");
            text = i18n.tr("You can now safely remove the device");
	    if (removeButton)
                removeButton.enabled = false
	    if (formatButton)
                formatButton.enabled = true
        }
        unmountOkButton.visible = true;
    } // onIsErrorChanged

    onDriveCountChanged: {
        if (driveCount == 0) {
            // TODO: really needed? 
            PopupUtils.close(confirmationDialog)
        }
    } // onDriveCountChanged
}
