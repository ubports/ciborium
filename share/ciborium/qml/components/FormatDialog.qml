import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Item {
    property int driveIndex

    height: childrenRect.height
    width: childrenRect.width

    Component {
        id: dialog

        Dialog {
            id: dialogue

            title: "Format"
            text: "This action will wipe the content from the device"

            Button {
                text: "Cancel"
                onClicked: PopupUtils.close(dialogue)
            }
            Button {
                text: "Continue with format"
                color: UbuntuColors.orange
                onClicked: {
                    driveCtrl.driveFormat(driveIndex)                     
                    PopupUtils.close(dialogue)
                }
            }
        }
    }
    Button {
        anchors.centerIn: parent
        id: formatButton
        text: "Format"
        onClicked: PopupUtils.open(dialog)
    }
}
