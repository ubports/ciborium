import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.ListItems 0.1 as ListItem

import "components"

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
            Component.onCompleted: driveCtrl.watch()

            ListView {
                model: driveCtrl.len
                anchors {
                    top: parent.top
                    bottom: parent.bottom
                    left: parent.left
                    right: parent.right
                    topMargin: units.gu(1)
                }

                delegate: UbuntuShape {
                    height: childrenRect.height
                    width: parent.width
                    color: index % 2 === 0 ? "#DECAE3" : "white"
                    anchors {
                        topMargin: units.gu(1)
                        bottomMargin: units.gu(1)
                    }

                    Column {
                        spacing: units.gu(2)
                        anchors {
                            leftMargin: units.gu(2)
                            topMargin: units.gu(1)
                            bottomMargin: units.gu(1)
                        }

                        Row {
                            spacing: units.gu(1)
                            height: units.gu(2)
                            width: childrenRect.width

                            Icon {
                                width: 24
                                height: 24
                                name: "media-memory-sd"
                                //source: "file:///usr/share/icons/Humanity/devices/48/media-memory-sd.svg"
                            }
                            Label {
                                width: paintedWidth       
                                text: driveCtrl.driveModel(index)
                            }
                        }
                        Row {
                            spacing: units.gu(1)
                            height: childrenRect.height
                            width: childrenRect.width
                            FormatDialog {
                                driveIndex: index
                            }
                            SafeRemoval {
                                driveIndex: index
                            }
                        }
                    }
                }
            }
        }
    }
}

