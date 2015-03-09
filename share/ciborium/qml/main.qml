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

	    Component {
                id: safeRemovalConfirmation
                SafeRemovalConfirmation {
                    id: safeRemovalConfirmationDialog
		    onButtonClicked: function() {
		        console.log("SafeRemovalDialog button clicked");
		        PopupUtils.close(safeRemovalConfirmationDialog)
                    }
		}
	    }

	    Component {
                id: safeRemoval
                SafeRemoval {
                    id: safeRemovalDialog
                    onCancelClicked: function() {
		        console.log("SafeRemoval cancelation button clicked");
		        PopupUtils.close(safeRemovalDialog);
	            }
                    onContinueClicked: function() {
		        console.log("SafeRemoval continue button clicked.")
                        driveCtrl.driveUnmount(safeRemovalDialog.driveIndex)
                        PopupUtils.close(safeRemovalDialog)
                        PopupUtils.open(safeRemovalConfirmation, mainPage)
		    }
                }
	    }

	    Component {
	    	id: formatConfirmation
		FormatConfirmation {
	    	    id: formatConfirmationDialog
		    onButtonClicked: function() {
		        console.log("FormatConfirmation button clicked");
		        PopupUtils.close(formatConfirmationDialog)
                    }
		}
	    }

	    Component {
	    	id: format
                FormatDialog {
	    	    id: formatDialog
                    onCancelClicked: function() {
		        console.log("Format cancelation button clicked");
		        PopupUtils.close(formatDialog);
	            }

                    onContinueClicked: function() {
		        console.log("Format continue button clicked.")
                        driveCtrl.driveFormat(formatDialog.driveIndex)                     
                        PopupUtils.close(formatDialog)
                        PopupUtils.open(formatConfirmation, mainPage)
		    }
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
                    color: index % 2 === 0 ? "white" : "#DECAE3"
                    anchors {
		        left: parent.left
			leftMargin: units.gu(1)
			right: parent.right
			rightMargin: units.gu(1)
                        topMargin: units.gu(1)
                        bottomMargin: units.gu(1)
                    }

		    DriveDelegate {
                        onFormatClicked: function() {
                            PopupUtils.open(format, mainPage, {"driveIndex": index})
			}

                        onSafeRemovalClicked: function() {
                            PopupUtils.open(safeRemoval, mainPage, {"driveIndex": index})
			}

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

