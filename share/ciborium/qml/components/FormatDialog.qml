import QtQuick 2.9
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.3

Dialog {
    id: formatDlg
    property int driveIndex

Button {
        id: okBtn
        text: i18n.tr("Continue with format")
        color: theme.palette.normal.negative
        onClicked: {
            switch(formatDlg.state) {
            case "confirm":
                console.log("Format confirmed");
                driveCtrl.driveFormat(formatDlg.driveIndex);
                d.confirmed = true;
                return;
            case "finished":
                console.log("Format completed");
                break;
            case "error":
                console.log("Error formatting!");
                break;
            default:
                console.warn("Ok button clicked in wrong state: ", formatDlg.state);
                break;
            }
            PopupUtils.close(formatDlg);
        }
    }
    
    Button {
        id: cancelBtn
        text: i18n.tr("Cancel")
        onClicked: {
            console.log("Format cancelled")
            PopupUtils.close(formatDlg)
        }
    }
    

    ActivityIndicator {
        id: formatActivity
        running: false
        visible:  running
    }

    state: "confirm"
    states: [
        State {
            name: "confirm"
            PropertyChanges {
                target: formatDlg
                explicit: true
                title: i18n.tr("Format")
                text: i18n.tr("This action will wipe the content from the device")
            }
        },
        State {
            name: "format"
            when: d.confirmed && driveCtrl.formatting && !driveCtrl.formatError
            PropertyChanges {
                target: formatDlg
                explicit: true
                title: i18n.tr("Formatting")
                text: i18n.tr("This action will wipe the content from the device")
            }
            PropertyChanges {
                target: cancelBtn
                visible: false
            }
            PropertyChanges {
                target: okBtn
                visible: false
            }
            PropertyChanges {
                target: formatActivity
                running: true
            }
        },
        State {
            name: "finish"
            when: d.confirmed && !driveCtrl.formatting && !driveCtrl.formatError
            PropertyChanges {
                target: formatDlg
                explicit: true
                title: i18n.tr("Format Complete")
                text: ""
            }
            PropertyChanges {
                target: cancelBtn
                visible: false
            }
            PropertyChanges {
                target: okBtn
                visible: true
                text: i18n.tr("Ok")
                color: theme.palette.normal.positive
            }
            PropertyChanges {
                target: formatActivity
                running: false
            }
        },
        State {
            name: "error"
            when: d.confirmed && driveCtrl.formatError
            PropertyChanges {
                target: formatDlg
                explicit: true
                title: i18n.tr("Format Error")
                text: i18n.tr("There was an error when formatting the device");
            }
            PropertyChanges {
                target: cancelBtn
                visible: false
            }
            PropertyChanges {
                target: okBtn
                visible: true
                text: i18n.tr("Ok")
                color: theme.palette.normal.overlaySecondaryText
            }
            PropertyChanges {
                target: formatActivity
                running: false
            }
        }
    ]

    QtObject {
        id: d
        property bool confirmed: false
    }
}
