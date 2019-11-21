import QtQuick 2.9
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.3
 
Dialog {
    id: safeRemovalDlg
    property int driveIndex
    
    Button {
        id: okButton
        text: i18n.tr("Continue")
        color: theme.palette.normal.positive
        onClicked: {
            switch (safeRemovalDlg.state) {
            case "remove":
                console.log("Continuing with safe removal");
                driveCtrl.driveUnmount(safeRemovalDlg.driveIndex);
                d.confirmed = true;
                return;
            case "finish":
                console.log("Safe removal complete");
                break;
            case "error":
                console.log("Error removing!");
                break;
            default:
                console.warn("Ok button clicked in wrong state: ", safeRemovalDlg.state);
                break;
            }
            PopupUtils.close(safeRemovalDlg);
        }
    }
    
    Button {
        id: cancelButton
        text: i18n.tr("Cancel")
        onClicked: {
            console.log("Safe removal action cancelled");
            PopupUtils.close(safeRemovalDlg);
        }
    }

    ActivityIndicator {
        id: unmountActivity
        running: false
        visible: running
    }

    state: "remove"
    states: [
        State {
            name: "remove"
            PropertyChanges {
                target: safeRemovalDlg
                explicit: true
                title: i18n.tr("Confirm remove")
                text: i18n.tr("Files on the device can't be accessed after removing")
            }
        },
        State {
            name: "unmount"
            when: d.confirmed && driveCtrl.unmounting && !driveCtrl.unmountError
            PropertyChanges {
                target: safeRemovalDlg
                explicit: true
                title: i18n.tr("Unmounting")
                text: ""
            }
            PropertyChanges {
                target: unmountActivity
                explicit: true
                visible: true
            }
            PropertyChanges {
                target: cancelButton
                explicit: true
                visible: false
            }
            PropertyChanges {
                target: okButton
                visible: false
                text: i18n.tr("Ok")
            }
        },
        State {
            name: "finish"
            when: d.confirmed && !driveCtrl.unmounting && !driveCtrl.unmountError
            PropertyChanges {
                target: safeRemovalDlg
                explicit: true
                title: i18n.tr("Safe to remove")
                text: i18n.tr("You can now safely remove the device");
            }
            PropertyChanges {
                target: okButton
                explicit: true
                visible: true
            }
            PropertyChanges {
                target: cancelButton
                explicit: true
                visible: false
            }
        },
        State {
            name: "error"
            when: d.confirmed && driveCtrl.unmountError
            PropertyChanges {
                target: safeRemovalDlg
                explicit: true
                title: i18n.tr("Unmount Error");
                text: i18n.tr("The device could not be unmounted because it is busy");
            }
            PropertyChanges {
                target: okButton
                explicit: true
                visible: true
                text: i18n.tr("Ok")
                color: theme.palette.normal.overlaySecondaryText
            }
            PropertyChanges {
                target: cancelButton
                explicit: true
                visible: false
            }
        }
    ]

    QtObject {
        id: d
        property bool confirmed: false
    }
}
