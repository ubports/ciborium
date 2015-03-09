import QtQuick 2.0
import Ubuntu.Components 1.1
import Ubuntu.Components.Popups 1.0

Dialog {
    property bool isError: driveCtrl.formatError
    property bool formatting: driveCtrl.formatting 
    property var onButtonClicked

    title: i18n.tr("Formatting")

    ActivityIndicator {
        id: formatActivity
        visible:  formatting && !isError
        running: formatting && !isError
        onRunningChanged: {
            if (!running && !isError) {
                PopupUtils.close(formattingDialog);
            }
        }
    }

    Button {
        id: okFormatErrorButton
        visible: false
        text: i18n.tr("Ok")
        color: UbuntuColors.orange
        onClicked: onButtonClicked()
    }

    onIsErrorChanged: {
        if (isError) {
            okFormatErrorButton.visible = true;
            formatActivity.visible = false;
            text= i18n.tr("There was an error when formatting the device");
        } else {
            okFormatErrorButton.visible= false;
            formatActivity.visible= true;
        }
    }

}
