import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Item {
    property int driveIndex

    height: childrenRect.height
    width: childrenRect.width

    Component {
        id: dialogFormat

        Dialog {
            id: dialogueFormat

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
                    PopupUtils.close(dialogueFormat)
                    PopupUtils.open(dialogFormatting)
                }
            }
        }
    }

    Component {
        id: dialogFormatting

        Dialog {
            id: dialogueFormatting

            title: i18n.tr("Formatting")

            ActivityIndicator {
                id: formatActivity
                running: driveCtrl.formatting
                onRunningChanged: {
                    if (!running) {
                        PopupUtils.close(dialogueFormatting);
                    }
                }
            }
        }
    }

    Button {
        anchors.centerIn: parent
        id: formatButton
        text: i18n.tr("Format")
        onClicked: PopupUtils.open(dialogFormat)
    }

    Component {
        id: dialogFormatError

        Dialog {
            id: dialogueFormatError

            title: i18n.tr("Format error")
            text: i18n.tr("There was an error when formatting the device.")

            ActivityIndicator {
                id: formatErrorActivity
                error: driveCtrl.formatError
                onRunningChanged: {
                    if (erro) {
                        PopupUtils.open(dialogueFormatError);
                    } else {
                        PopupUtils.close(dialogueFormatError);
		    }
                }
            }

            Button {
                text: i18n.tr("Ok")
                color: UbuntuColors.orange
                onClicked: {
                    PopupUtils.close(dialogueFormatError)
                }
            }
        }
    }
}
