import QtQuick 2.9
import QtQuick.Layouts 1.3
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.1

ListItem {
    id: driveDelegate
    property int driveIndex

    signal formatClicked()
    signal safeRemovalClicked()

    width: parent.width
    height: layout.implicitHeight
    expansion.height: layout.implicitHeight + buttonRow.height + units.gu(3)
    onClicked: expansion.expanded = !expansion.expanded

    ListItemLayout {
        id: layout
        title.text: driveCtrl.driveModel(index)

        Icon {
            height: units.gu(4)
            width: height
            anchors {
                left: parent.left
                leftMargin: units.gu(2)
                top: parent.top
                topMargin: units.gu(2)
            }
            source: Qt.resolvedUrl("../../icons/memory-card.svg")
            SlotsLayout.position: SlotsLayout.Leading
        }
    }

    RowLayout {
        id: buttonRow
        anchors {
            left: parent.left
            top: layout.bottom
            right: parent.right
            margins: units.gu(1)
        }
        height: units.gu(3)
        spacing: units.gu(1)

        // Spacer to force the buttons to the right side
        // using Layout.alignment causes weird spacing...
        Item {
            Layout.fillWidth: true
        }

        Button {
            text: i18n.tr("Format")
            onClicked: formatClicked()
        }

        Button {
            text: i18n.tr("Safely Remove")
            color: theme.palette.selected.focus
            onClicked: safeRemovalClicked()
        }
    }
}
