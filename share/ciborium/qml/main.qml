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

	    // dialogs required to show progress information to the user
	    Component {
                id: safeRemovalConfirmationDialog
                SafeRemovalConfirmation {
                    confirmationDialog: safeRemovalConfirmationDialog
		}
	    }

	    Component {
                id: safeRemovalDialog
                SaveRemoval {
                    parentWindow: mainPage
                    removalDialog: safeRemovalDialog
                    confirmationDialog: safeRemovalConfirmationDialog
                }
	    }

	    Component {
	    	id: formatConfirmationDialog
		FormatConfirmation {
                    formattingDialog: formatConfirmationDialog
		}
	    }

	    Component {
	    	id: formatDialog
                FormatDialog {
                    parentWindow: mainPage
                    formatDialog: formatDialog
                    formattingDialog: formatConfirmationDialog
		}
	    }


            ListView {
                model: driveCtrl.len
                spacing: units.gu(1)

                anchors {
                    top: parent.top
                    bottom: parent.bottom
                    left: parent.left
                    right: parent.right
                    topMargin: units.gu(1)
                } // anchors

                delegate: UbuntuShape {
                    height: childrenRect.height
                    width: parent.width
                    color: index % 2 === 0 ? "white" : "#DECAE3"
                    anchors {
                        topMargin: units.gu(1)
                        bottomMargin: units.gu(1)
                    }

		    DriveDeleage {
                        driverIndex: index
                        parentWindow: mainPage
                        formatDialog: formatDialog
                        removeDialog: safeRemovalDialog

                        anchors {
                            leftMargin: units.gu(2)
                            topMargin: units.gu(1)
                            bottomMargin: units.gu(1)
                        }
		    }

                } // delegate
            } // ListView
        } // Page
    }
}

