import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Dialog {
    property int driveIndex
    property var parentWindow
    property var formatDialog
    property var formattingDialog

    title: i18n.tr("Format")
    text: i18n.tr("This action will wipe the content from the device")

    Button {
        text: i18n.tr("Cancel")
        onClicked: PopupUtils.close(dialogueFormat)
    }
    Button {
        text: i18n.tr("Continue with format")
        color: UbuntuColors.orange
        onClicked: {
            driveCtrl.driveFormat(driveIndex)                     
            PopupUtils.close(formatDialog, parentWindow)
            PopupUtils.open(formattingDialog, parentWindow)
        }
    }
}
