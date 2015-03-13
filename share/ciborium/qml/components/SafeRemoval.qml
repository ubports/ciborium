import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0


Dialog {
    property int driveIndex
    property var formatButton
    property var removeButton
    property var onCancelClicked
    property var onContinueClicked

    title: i18n.tr("Confirm remove")
    text: i18n.tr("Files on the device can't be accessed after removing")

    Button {
        text: i18n.tr("Cancel")
        onClicked: onCancelClicked(formatButton, removeButton) 
    } // Button Cancel

    Button {
        text: i18n.tr("Continue")
        color: UbuntuColors.orange
        onClicked: onContinueClicked(formatButton, removeButton)
   }
}
