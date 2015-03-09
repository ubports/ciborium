import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0


Dialog {
    property int driverIndex
    property var removalDialog
    property var confirmationDialog

    title: i18n.tr("Confirm remove")
    text: i18n.tr("Files on the device can't be accessed after removing")

    Button {
        text: i18n.tr("Cancel")
        onClicked: PopupUtils.close(id)
    } // Button Cancel

    Button {
        text: i18n.tr("Continue")
        color: UbuntuColors.orange
        onClicked: {
            driveCtrl.driveUnmount(driverIndex)
            PopupUtils.close(removalDialog)
            saferemoval.enabled= false
            PopupUtils.open(unmountinDialog)
        }
   }
}
