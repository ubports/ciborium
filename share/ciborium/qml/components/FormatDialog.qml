import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Dialog {
    property int driveIndex
    property var onCancelClicked
    property var onContinueClicked

    title: i18n.tr("Format")
    text: i18n.tr("This action will wipe the content from the device")

    Button {
        text: i18n.tr("Cancel")
        onClicked: onCancelClicked()
    }
    Button {
        text: i18n.tr("Continue with format")
        color: UbuntuColors.orange
        onClicked: onContinueClicked()
    }
}
