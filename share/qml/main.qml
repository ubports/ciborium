import QtQuick 2.0
import Ubuntu.Components 0.1

/*!
    \brief MainView with a Label and Button elements.
*/

MainView {
    // objectName for functional testing purposes (autopilot-qt5)
    objectName: "mainView"

    // Note! applicationName needs to match the "name" field of the click manifest
    applicationName: "ciborium"

    /*
     This property enables the application to change orientation
     when the device is rotated. The default is false.
    */
    //automaticOrientation: true

    width: units.gu(100)
    height: units.gu(75)

    PageStack {
        id: stack
        Component.onCompleted: push(mainPage)
        
        Page{
            id: mainPage
            title: i18n.tr("SD Card Management")

            Column {
                spacing: units.gu(1)
                anchors {
                    margins: units.gu(2)
                    fill: parent
                }

                Label {
                    id: label
                    objectName: "label"

                    text: "drives: \"" + driveCtrl.ExternalDrives + "\""
                }

                Button {
                    objectName: "button"
                    width: parent.width

                    text: i18n.tr("Refresh")

                    onClicked: driveCtrl.drives()
                }
            }
        }
    }
}

