import QtQuick 2.9
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.1

import "./components"

MainView {
    id: mainView
    // Note! applicationName needs to match the "name" field of the click manifest
    applicationName: "ciborium"

    width: units.gu(100)
    height: units.gu(75)

    Page {
        id: mainPage
        anchors.fill: parent

        header: PageHeader {
            id: ph
            title: i18n.tr("SD Card Management")
        }

        ListView {
            anchors {
                top: ph.bottom
                left: parent.left
                right: parent.right
                bottom: parent.bottom
            }
            spacing: units.gu(1)
            model: driveCtrl.len
            delegate: DriveDelegate {
                driveIndex: index
                onFormatClicked: {
                    console.log("Format button clicked")
                    PopupUtils.open(Qt.resolvedUrl("./components/FormatDialog.qml", mainPage, {"driveIndex": index}))
                }
                onSafeRemovalClicked: {
                    console.log("Safe removal button clicked")
                    PopupUtils.open(Qt.resolvedUrl("./components/SafeRemoval.qml", mainPage, {"driveIndex": index}))
                }
            }
        }
        Component.onCompleted: driveCtrl.watch()
    }
}

