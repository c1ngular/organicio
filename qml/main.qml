import QtQuick 2.6
import QtQuick.Controls 2.2
import QtQuick.Window 2.2
import QtQuick.Controls.Material 2.12
import QtQuick.Controls.Universal 2.12
import QtQuick.Layouts 1.3
import QtGraphicalEffects 1.12



import mpv 1.0

Item {
    property var previewVideoPadding:5
    property var previewVideoHeight:(Screen.height * 0.1)
    property var previewBarPadding:20
    property var previewBarHeight: (previewVideoHeight + previewVideoPadding * 2 + previewBarPadding * 2)
    property var previewVideoRatio: (16 / 9)
    property var streamingIndicatorRadius: 10
    property var infoConWidth: (Screen.width * 0.28)
    property var msgConPadding:10
    property var messageBoxHeight:(Screen.height * 0.24) 
    property var controlBoxMessageBoxGap:20
    property var sysStatusBarHeightRatio: (1/8)
    property var sensorConPadding: 20
    property var controlConPadding:20
    property var sensorNumEachRow:5
    property var sensorNumEachColumn:3
    property var universalBorderRadius:5
    property var sensorControlGap:0
    property var infoPreviewGap:20
    property var previewVideoBorderWidth:2
    property var sensorConBorderWidth:1
    property var controlConBorderWidth:1
    property var sysInfoControlSegHeight:1


    property var sensorInfoIconHeightRatio:(2 / 3)
    property var sensorInfoValueHeightRatio:(1 / 3)
    property var sensorIconParentHeightPercent:0.7
    property var sensorTextParentHeightPercent:0.5

    property var sysInfoIconParentWidthPercent:0.3
    property var sysInfoTextParentWidthPercent:0.2
    property var sysInfoTextColor:Material.color(Material.Orange)
    property var sysInfoTextLetterSpace:1
    property var sysInfoTextWordSpace:5

    property var previewBarBgColor:Qt.rgba(0, 0, 0, 0.8)
    property var sensorConBgColor:Qt.rgba(0, 0, 0, 0.4)
    property var controlConBgColor:Qt.rgba(0, 0, 0, 0.4)
    property var sensorConBorderColor:Qt.rgba(1, 1, 1, 0.8)
    property var controlConBorderColor:Qt.rgba(1, 1, 1, 0.8)
    property var controlScrollPadding:10
    property var controlEachItemMargin:20
    property var controlEachOptionHeightBase:80
    property var controlItemSectionTitleColor:"white"


    property var videoInactiveBorderColor:"#555"
    property var videoPreviewingBorderColor:Material.color(Material.Green)
    property var videoInactiveBgColor:"black"
    property var videoStreamingBgColor:Material.color(Material.Red)
    property var videoStreamingAnimationColor:Material.color(Material.Red)
    property var sensorInfoTextColor:"white"
    property var sensorInfoTextLetterSpace:1
    property var sensorInfoTextWordSpace:5
    property var sensorInforGlowColor:"#ccc"
    property var sensorInfoGlowRadius:10
    property var sensorInfoGlowSample:20
    property var sensorInfoGlowSpread:0
   
    

    property var siteName:"丽江老君山蓝莓谷"
    property var siteGps:"GPS：100.2356,26.8740"
    property var siteUptime:"UPtime：15 星期 6 天 23 小时"

    property var loadingIndicatorSize:80

    function switchPreview(index,url){

        var currentPrevIndex

        for(var i = 0; i < streamModel.count; i++) {
            if(streamModel.get(i).previewing == 1){
                currentPrevIndex=i
            }
        }

        if(currentPrevIndex !== index){
            streamModel.get(currentPrevIndex).previewing = 0
            streamModel.get(index).previewing = 1
            mainvideo.command(["loadfile", url])
        }
        
    }

    function switchStreaming(index,url){

        var currentStreamingIndex

        for(var i = 0; i < streamModel.count; i++) {
            if(streamModel.get(i).streaming == 1){
                currentStreamingIndex=i
            }
        }

        if(currentStreamingIndex !== index){
            streamModel.get(currentStreamingIndex).streaming = 0
            streamModel.get(index).streaming = 1
        }
        
    }

    function togglePreviews(){
        if(previewsCon.state=="hidePreview"){
            previewsCon.state="showPreview"
        }else{
            previewsCon.state="hidePreview"
        }

    }

    function showPreviews(){
        previewsCon.state="showPreview"
    }

    function hidePreviews(){
        previewsCon.state="hidePreview"
    }    

    function toggleSensors(){
        if(sensorCon.state=="hideSensor"){
            sensorCon.state="showSensor"
        }else{
            sensorCon.state="hideSensor"
        }
    }

    function showSensors(){
        sensorCon.state="showSensor"
    }

    function hideSensors(){
        sensorCon.state="hideSensor"
    }

    function toggleControls(){
        if(controlCon.state=="hideControl"){
            controlCon.state="showControl"
        }else{
            controlCon.state="hideControl"
        }
    }

    function showControls(){
        controlCon.state="showControl"
    }

    function hideControls(){
        controlCon.state="hideControl"
    }

    function formatDateTime(){
        var m = new Date();
        var days=["星期天","星期一","星期二","星期三","星期四","星期五","星期六"]
        var dateString =
            m.getUTCFullYear() + "年" +
            ("0" + (m.getMonth()+1)).slice(-2) + "月" +
            ("0" + m.getDate()).slice(-2) + "日" +
            ("0" + m.getHours()).slice(-2) + ":" +
            ("0" + m.getMinutes()).slice(-2) + ":" +
            ("0" + m.getSeconds()).slice(-2) + "  " + 
            days[m.getDay()]
        return dateString
    }

    property var defaultMainVideoVolume:50

    function setMainVideoPlayerVolume(volume){
        mainvideo.setProperty("volume",volume)
    }

    function showLoadingIndicator(){
        loadingIndicator.running=true
        loadingPopUp.open()
    }

    function hideLoadingIndicator(){
        loadingIndicator.running=false
        loadingPopUp.close()
    }    

    ListModel {
        id:systemInfoModel

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/cpu.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/gpu.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/ctemp.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/memo.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/disk.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }

        ListElement {
            iname: "CPU"
            iicon: "qrc:/qml/icons/internet.svg"
            iuid:"temp"
            ivlu:"36"
            iunit:"℃"
            threshold:""
        }                        
      
    }

    ListModel {
        id:sensorModel

        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/temp.svg"
            suid:"temp"
            suid_duplica:"s6"
            svlu:"36"
            sunit:"℃"
        }

        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/humidity.svg"
            suid:"humidity"
            suid_duplica:"s4"
            svlu:"67"
            sunit:"%"
        }
        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/winddirection.svg"
            suid:"winddirection"
            suid_duplica:"s3"
            svlu:"东南"
            sunit:""
        }

        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/windspeed.svg"
            suid:"windspeed"
            suid_duplica:"s1"
            svlu:"12"
            sunit:"级"
        }

       ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/temp.svg"
            suid:"temp2"
            suid_duplica:"s62"
            svlu:"36"
            sunit:"℃"
        }

        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/humidity.svg"
            suid:"humidity2"
            suid_duplica:"s42"
            svlu:"67"
            sunit:"%"
        }
        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/winddirection.svg"
            suid:"winddirection2"
            suid_duplica:"s32"
            svlu:"东南"
            sunit:""
        }

        ListElement {
            sname: "Jim Williams"
            sicon: "qrc:/qml/icons/windspeed.svg"
            suid:"windspeed2"
            suid_duplica:"s12"
            svlu:"12"
            sunit:"级"
        }        
      
    }

    ListModel {
        id: streamModel

        ListElement {
            uid: "Apple"
            uid_duplica:"x"
            url:"rtmp://hwzbout.yunshicloud.com/mj1170/h6f7wv"
            streaming:1
            previewing:1
        }
        ListElement {
            uid: "Orange"
            uid_duplica:"y"
            url: "rtmp://202.69.69.180:443/webcast/bshdlive-pc"
            streaming:0
            previewing:0
        }

        ListElement {
            uid: "tApptle"
            uid_duplica:"xf"
            url:"rtmp://hwzbout.yunshicloud.com/mj1170/h6f7wv"
            streaming:0
            previewing:0
        }
        ListElement {
            uid: "Oradfnge"
            uid_duplica:"yes"
            url: "rtmp://202.69.69.180:443/webcast/bshdlive-pc"
            streaming:0
            previewing:0
        }
   
    }

    Rectangle{
        id:mainCon
        anchors.fill: parent

        MpvPlayer {
            id: mainvideo
            anchors.fill: parent   
                 
        }
    }

    Rectangle {
        id: mainLayout
        anchors.fill: mainCon
        color:"transparent"

        ColumnLayout{
            id:mainLayoutSectioner
            anchors.fill: parent
            spacing:infoPreviewGap

            Rectangle {
                id:infoCon
                color: "transparent"
                Layout.fillWidth: true
                height:Screen.height - previewBarHeight - infoPreviewGap
                RowLayout{
                    anchors.fill: parent
                    spacing: sensorControlGap
                    Rectangle{
                        id:sensorSwipable
                        color:"transparent"
                        width:Screen.width - infoConWidth - sensorControlGap
                        Layout.fillHeight: true
                        MouseArea {
                            anchors.fill: parent
                            onClicked: {
                                toggleSensors()
                            }
                        }  
                        Rectangle{
                            id:sensorCon
                            color: "transparent"
                            width:parent.width
                            height:parent.height
                            x: parent.width * -1
                            y:0
                            states:[
                                State {
                                    name: "showSensor"
                                    PropertyChanges { target: sensorCon; x:0}
                                },
                                State {
                                    name: "hideSensor"
                                    PropertyChanges { target: sensorCon; x:parent.width * -1}
                                }
                            ]

                            transitions: [
                                Transition {
                                    to: "showSensor"
                                    NumberAnimation { properties: "x"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                                },
                                Transition {
                                    to: "hideSensor"
                                    NumberAnimation { properties: "x"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                                }
                            ]

                            Component.onCompleted: {
                                showSensors()
                            }

                            Component {
                                id: sensorDelegate
                                Rectangle {
                                    width: sensors.cellWidth 
                                    height: sensors.cellHeight
                                    color:"transparent"
                                    Rectangle{
                                        width: parent.width - sensorConPadding * 2 
                                        height: parent.height - sensorConPadding * 2
                                        color:"transparent"
                                        anchors.centerIn:parent
                                        Column {
                                            anchors.fill:parent
                                            spacing:0
                                            Rectangle{
                                                width:parent.width
                                                color:"transparent"
                                                height:parent.height * sensorInfoIconHeightRatio
                                                Image{
                                                    id:suid
                                                    height: parent.height * sensorIconParentHeightPercent 
                                                    anchors.centerIn:parent
                                                    source:sicon
                                                    fillMode: Image.PreserveAspectFit
                                                }
                                                Glow {
                                                    anchors.fill: suid
                                                    radius: sensorInfoGlowRadius
                                                    samples: sensorInfoGlowSample
                                                    color: sensorInforGlowColor
                                                    spread:sensorInfoGlowSpread
                                                    source: suid
                                                
                                                }
                                            }

                                            Rectangle{
                                                width:parent.width
                                                color:"transparent"
                                                height:parent.height * sensorInfoValueHeightRatio
                                                Text{
                                                    id: suid_duplica
                                                    anchors.centerIn:parent
                                                    color: sensorInfoTextColor
                                                    text: svlu + sunit
                                                    clip:true
                                                    font{
                                                        letterSpacing:sensorInfoTextLetterSpace
                                                        wordSpacing:sensorInfoTextWordSpace
                                                        pointSize: parent.height * sensorTextParentHeightPercent
                                                    }
                                                }

                                                Glow {
                                                    anchors.fill: suid_duplica
                                                    radius: sensorInfoGlowRadius
                                                    samples: sensorInfoGlowSample
                                                    color: sensorInforGlowColor
                                                    spread:sensorInfoGlowSpread
                                                    source: suid_duplica
                                                
                                                }
                                            }
                                        }
                                    }
                                }
                            }

                            Component {
                                id: siteDesc
                                Rectangle {
                                    width: sensors.width
                                    height: sensors.cellHeight
                                    color:"transparent"
                                    Rectangle{
                                        width:parent.width - sensorConPadding * 2 
                                        height:parent.height - sensorConPadding * 2
                                        color:"transparent"
                                        anchors.centerIn:parent 
                                        Column{
                                            spacing:0
                                            anchors.fill:parent
                                            Rectangle{
                                                width:parent.width 
                                                height:parent.height / 5
                                                color:"transparent"
                                                Timer {
                                                    interval: 200
                                                    running: true
                                                    repeat: true
                                                    onTriggered:currentDateTime.text=formatDateTime()
                                                }

                                                Text{
                                                    id:currentDateTime
                                                    anchors{
                                                        right:parent.right
                                                        verticalCenter:parent.verticalCenter
                                                    }
                                                    color: sensorInfoTextColor
                                                    text: ""
                                                    clip:true
                                                    font{
                                                        letterSpacing:sensorInfoTextLetterSpace
                                                        wordSpacing:sensorInfoTextWordSpace
                                                        pointSize:parent.height * sensorTextParentHeightPercent * 0.7
                                                    }
                                                }   

                                                Glow {
                                                    anchors.fill: currentDateTime
                                                    radius: sensorInfoGlowRadius
                                                    samples: sensorInfoGlowSample
                                                    color: sensorInforGlowColor
                                                    spread:sensorInfoGlowSpread
                                                    source: currentDateTime
                                                
                                                }
                                            }
                                            Rectangle{
                                                width:parent.width 
                                                height:parent.height / 5 * 4
                                                color:"transparent"
                                                Column{
                                                    spacing:0
                                                    anchors.fill:parent
                                                    Rectangle{
                                                        width:parent.width
                                                        height:parent.height / 3 * 2
                                                        color:"transparent"
                                                        Text{
                                                            id:siteNameView
                                                            anchors.centerIn:parent
                                                            color: sensorInfoTextColor
                                                            text: siteName
                                                            clip:true
                                                            font{
                                                                letterSpacing:sensorInfoTextLetterSpace
                                                                wordSpacing:sensorInfoTextWordSpace
                                                                pointSize:parent.height * sensorTextParentHeightPercent
                                                            }
                                                        }   
                                                        Glow {
                                                            anchors.fill: siteNameView
                                                            radius: sensorInfoGlowRadius
                                                            samples: sensorInfoGlowSample
                                                            color: sensorInforGlowColor
                                                            spread:sensorInfoGlowSpread
                                                            source: siteNameView
                                                        
                                                        }                                                       
                                                    }
                                                    Rectangle{
                                                        width:parent.width
                                                        height:parent.height / 3
                                                        color:"transparent"
                                                        Row{
                                                            spacing:0
                                                            anchors.fill:parent
                                                            Rectangle{
                                                                width:parent.width / 2
                                                                height:parent.height
                                                                color:"transparent"
                                                                Text{
                                                                    id:siteGpsView
                                                                    anchors.centerIn:parent
                                                                    color: sensorInfoTextColor
                                                                    text: siteGps
                                                                    clip:true
                                                                    font{
                                                                        letterSpacing:sensorInfoTextLetterSpace
                                                                        wordSpacing:sensorInfoTextWordSpace
                                                                        pointSize:parent.height * sensorTextParentHeightPercent
                                                                    }
                                                                } 

                                                                Glow {
                                                                    anchors.fill: siteGpsView
                                                                    radius: sensorInfoGlowRadius
                                                                    samples: sensorInfoGlowSample
                                                                    color: sensorInforGlowColor
                                                                    spread:sensorInfoGlowSpread
                                                                    source: siteGpsView
                                                                
                                                                }     

                                                            }
                                                            Rectangle{
                                                                width:parent.width / 2
                                                                height:parent.height
                                                                color:"transparent"
                                                                Text{
                                                                    id:siteUptimeView
                                                                    anchors.centerIn:parent
                                                                    color: sensorInfoTextColor
                                                                    text: siteUptime
                                                                    clip:true
                                                                    font{
                                                                        letterSpacing:sensorInfoTextLetterSpace
                                                                        wordSpacing:sensorInfoTextWordSpace
                                                                        pointSize:parent.height * sensorTextParentHeightPercent
                                                                    }
                                                                }   

                                                                Glow {
                                                                    anchors.fill: siteUptimeView
                                                                    radius: sensorInfoGlowRadius
                                                                    samples: sensorInfoGlowSample
                                                                    color: sensorInforGlowColor
                                                                    spread:sensorInfoGlowSpread
                                                                    source: siteUptimeView
                                                                
                                                                }                                                                                                                                        
                                                            }
                                                        }                                                    
                                                    }
                                                }                                                   
                                            }
                                        }
                                    }
                                }
                            }

                            Rectangle{
                                width:parent.width - sensorConPadding * 2
                                height:parent.height - sensorConPadding * 2
                                anchors.centerIn:parent
                                color:sensorConBgColor
                                radius:universalBorderRadius
                                border{
                                    color:sensorConBorderColor
                                    width:sensorConBorderWidth
                                }
                                GridView {
                                    clip:true
                                    id:sensors
                                    anchors.fill:parent
                                    cellWidth: parent.width / sensorNumEachRow
                                    cellHeight:parent.height / sensorNumEachColumn
                                    model: sensorModel
                                    delegate: sensorDelegate
                                    header:siteDesc
                                    focus:false
                                }                            
                            }
                            
                        }
                    }

                    Rectangle{
                        id:controlSwipable
                        color:"transparent"
                        width:infoConWidth
                        Layout.fillHeight: true
                        MouseArea {
                            anchors.fill: parent
                            onClicked: {
                                toggleControls()
                            }
                        }  
                        Rectangle{
                            id:controlCon
                            color: "transparent"
                            width:parent.width
                            height:parent.height
                            x: infoConWidth
                            y:0
                            states:[
                                State {
                                    name: "showControl"
                                    PropertyChanges { target: controlCon; x:0}
                                },
                                State {
                                    name: "hideControl"
                                    PropertyChanges { target: controlCon; x:infoConWidth}
                                }
                            ]

                            transitions: [
                                Transition {
                                    to: "showControl"
                                    NumberAnimation { properties: "x"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                                },
                                Transition {
                                    to: "hideControl"
                                    NumberAnimation { properties: "x"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                                }
                            ]

                            Component.onCompleted: {
                                showControls()
                            }

                            Column{
                                spacing:controlBoxMessageBoxGap
                                width:parent.width - controlConPadding * 2
                                height:parent.height - controlConPadding * 2
                                anchors.centerIn:parent
                                Rectangle{
                                    width:parent.width
                                    height:parent.height - messageBoxHeight
                                    color:controlConBgColor   
                                    radius:universalBorderRadius
                                    border{
                                        color:controlConBorderColor
                                        width:controlConBorderWidth
                                    }
                                    Column{
                                        spacing:0
                                        anchors.fill:parent
                                        Rectangle{
                                            width:parent.width
                                            height: parent.height * sysStatusBarHeightRatio
                                            color:"transparent"
                                            GridView {
                                                clip:true
                                                id:systemWatch
                                                anchors.fill:parent
                                                cellWidth: parent.width / systemInfoModel.count
                                                cellHeight:parent.height
                                                model: systemInfoModel
                                                delegate: systemInfoDelegate
                                                focus:false
                                            }   
                                            Component{
                                                id:systemInfoDelegate
                                                Rectangle{
                                                    width:systemWatch.cellWidth
                                                    height:systemWatch.cellHeight
                                                    color:"transparent"
                                                    Column{
                                                        spacing:0
                                                        anchors.fill:parent
                                                        Rectangle{
                                                            width:parent.width
                                                            height:parent.height / 2
                                                            color:"transparent"
                                                            Image{
                                                                height: parent.width * sysInfoIconParentWidthPercent 
                                                                anchors.centerIn:parent
                                                                source:iicon
                                                                fillMode: Image.PreserveAspectFit
                                                            }
                                                        }
                                                        Rectangle{
                                                            width:parent.width
                                                            height:parent.height / 2
                                                            color:"transparent"
                                                            Text{
                                                                anchors.centerIn:parent
                                                                color: sysInfoTextColor
                                                                text: ivlu+iunit
                                                                clip:true
                                                                font{
                                                                    letterSpacing:sysInfoTextLetterSpace
                                                                    wordSpacing:sysInfoTextWordSpace
                                                                    pointSize: parent.width * sysInfoTextParentWidthPercent
                                                                }
                                                            }                                                        
                                                        }
                                                    }
                                                }
                                            }                                                                            
                                        }
                                        Rectangle{
                                            width:parent.width
                                            height:sysInfoControlSegHeight
                                            color:controlConBorderColor
                                        }
                                        Rectangle{
                                            id:controlsBox
                                            width:parent.width
                                            height:parent.height * (1 - sysStatusBarHeightRatio) - sysInfoControlSegHeight
                                            color:"transparent"
                                            ScrollView {
                                                id:controlScroller
                                                width:parent.width - controlScrollPadding * 2
                                                height:parent.height - controlScrollPadding * 2 
                                                anchors.centerIn: parent
                                                ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                                                ScrollBar.vertical.policy: ScrollBar.AlwaysOn
                                                clip:true

                                                Column{
                                                    spacing:controlEachItemMargin
                                                    width:parent.width
                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "本地音量"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Slider {
                                                                id: volumeSetter
                                                                value: defaultMainVideoVolume
                                                                from:0
                                                                to:100
                                                                width: parent.width
                                                                anchors.centerIn:parent
                                                                onMoved:{
                                                                    setMainVideoPlayerVolume(volumeSetter.value)
                                                                }
                                                            }                                               
                                                        }                                                         
                                                    }   
                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 2
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "背景音乐"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                Switch {
                                                                    text: "背景音乐开关"
                                                                }
                                                                Slider {
                                                                    id: bgMusicVolume
                                                                    value: 0.5
                                                                    width: parent.width
                                                                }                                                               
                                                            }                                             
                                                        }                                                         
                                                    }         

                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 2
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "水印图片"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                Switch {
                                                                    text: "水印图片开关"
                                                                }
                                                                ComboBox {
                                                                    model: ["左上角", "左下角", "右上角","右下角"]
                                                                    width: parent.width
                                                                }                                                            
                                                            }                                             
                                                        }                                                         
                                                    }         

                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 3
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "视频推送尺寸"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                RadioButton {
                                                                    text: "720p"
                                                                }
                                                                RadioButton {
                                                                    text: "480p"
                                                                }
                                                                RadioButton {
                                                                    text: "320p"
                                                                }                                                          
                                                            }                                             
                                                        }                                                         
                                                    }   

                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 3
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "视频推送质量"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                RadioButton {
                                                                    text: "高清"
                                                                }
                                                                RadioButton {
                                                                    text: "标准"
                                                                }
                                                                RadioButton {
                                                                    text: "低清"
                                                                }                                                          
                                                            }                                             
                                                        }                                                         
                                                    }       

                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 1.2
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "视频推送码率(KB)"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color: controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                SpinBox {
                                                                    id: box
                                                                    from:200
                                                                    to:5000
                                                                    stepSize:100
                                                                    value: 1000
                                                                    width: parent.width
                                                                    anchors.horizontalCenter: parent.horizontalCenter
                                                                    editable: false
                                                                }                                                        
                                                            }                                             
                                                        }                                                         
                                                    }       

                                                    Rectangle{
                                                        width:parent.width
                                                        height:controlEachOptionHeightBase * 2
                                                        color:"transparent"
                                                        GroupBox {
                                                            title: "传感器"
                                                            anchors.fill:parent
                                                            label: Label {
                                                                width: parent.width
                                                                text: parent.title
                                                                color:controlItemSectionTitleColor
                                                                elide: Text.ElideRight
                                                            }
                                                            Column{
                                                                spacing:controlEachItemMargin
                                                                width:parent.width
                                                                Switch {
                                                                    text: "写入输出画面"
                                                                }      
                                                                ComboBox {
                                                                    model: ["顶部", "中间", "底部"]
                                                                    width: parent.width
                                                                }                                                                                                              
                                                            }                                             
                                                        }                                                         
                                                    }                                                                                                                                                                                                                                                                                                                                                                                                                                              
                                                }                          
                                            }
                                        }
                                    }     
                                }
                                Rectangle{
                                    width:parent.width
                                    height:messageBoxHeight - controlBoxMessageBoxGap
                                    color:"transparent"
                                    radius:universalBorderRadius
                                    border{
                                        color:controlConBorderColor
                                        width:controlConBorderWidth
                                    }
                                    
                                    Column{
                                        spacing:msgConPadding
                                        padding:msgConPadding
                                        width:parent.width - msgConPadding * 2
                                        height:parent.height - msgConPadding * 2
                                        TextArea {
                                            width: parent.width
                                            height:(parent.height - msgConPadding) * 0.7
                                            wrapMode: TextArea.Wrap
                                            placeholderText: "输入心晴日志..."
                                            placeholderTextColor :"#ccc"
                                        }
                                        Button {
                                            id: msgbutton
                                            text: "发送"
                                            width:parent.width * 0.5
                                            height: (parent.height - msgConPadding) * 0.3
                                            highlighted: true
                                        }
                                    }
                                }
                            }
                            
                        }                        
                    }                
                }
            }

            Rectangle{

                id:previewSwipable
                Layout.fillWidth: true
                height:previewBarHeight
                color:"transparent"
                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        togglePreviews()
                    }
                }  
                Rectangle {
                    id:previewsCon
                    color: previewBarBgColor
                    width:parent.width
                    height:parent.height
                    x:0
                    y:previewBarHeight
                    states:[
                        State {
                            name: "showPreview"
                            PropertyChanges { target: previewsCon; y:0}
                        },
                        State {
                            name: "hidePreview"
                            PropertyChanges { target: previewsCon; y:previewBarHeight}
                        }
                    ]

                    transitions: [
                        Transition {
                            to: "showPreview"
                            NumberAnimation { properties: "y"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                        },
                        Transition {
                            to: "hidePreview"
                            NumberAnimation { properties: "y"; easing.type: Easing.InOutQuad; duration: 400; loops: 1 }
                        }
                    ]

                    Rectangle{
                        id:previews
                        color:"transparent"
                        width:Screen.width - previewBarPadding * 2
                        height:previewBarHeight - previewBarPadding * 2
                        anchors.centerIn : parent

                        ScrollView {
                            anchors.fill: parent
                            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                            ScrollBar.vertical.policy: ScrollBar.AlwaysOff
                            clip:true
                            Component {
                                id: streamingDelegate
                                Rectangle {
                                    height:previewVideoHeight + previewVideoPadding * 2
                                    width:previewVideoHeight * previewVideoRatio + previewVideoPadding * 2
                                    color: streaming == 1 ? videoStreamingBgColor : videoInactiveBgColor
                                    radius:universalBorderRadius
                                    border{
                                        color: previewing == 1 ? videoPreviewingBorderColor : videoInactiveBorderColor
                                        width: previewVideoBorderWidth
                                    }

                                    MpvPlayer {
                                        id: uid
                                        height:previewVideoHeight
                                        width:previewVideoHeight * previewVideoRatio 
                                        anchors.centerIn:parent
                                        Component.onCompleted: {
                                            uid.command(["loadfile", url])
                                            uid.setProperty("mute", "yes")
                                        }
                                    
                                    }                                   

                                    Rectangle{
                                        id: uid_duplica
                                        width:streamingIndicatorRadius
                                        height:streamingIndicatorRadius
                                        radius:streamingIndicatorRadius / 2 
                                        color: streaming == 1 ? videoStreamingAnimationColor : "transparent"
                                        anchors.right:parent.right
                                        anchors.bottom:parent.bottom
                                        anchors.rightMargin: streamingIndicatorRadius
                                        anchors.bottomMargin:streamingIndicatorRadius    

                                    }

                                    SequentialAnimation {
                                        ParallelAnimation {

                                            OpacityAnimator {
                                                target: uid_duplica
                                                from: 0.2
                                                to: 1
                                                easing.type: Easing.Linear;
                                                duration: 500
                                            }

                                            ScaleAnimator {
                                                target: uid_duplica
                                                from: 0.2
                                                to: 1
                                                easing.type: Easing.OutExpo;
                                                duration: 500
                                            }
                                        }

                                        ParallelAnimation {

                                            OpacityAnimator {
                                                target: uid_duplica
                                                from: 1
                                                to: 0.2
                                                easing.type: Easing.Linear;
                                                duration: 500
                                            }

                                            ScaleAnimator {
                                                target: uid_duplica
                                                from: 1
                                                to:0.2
                                                easing.type: Easing.Linear;
                                                duration: 500
                                            }
                                        }
                                        PauseAnimation { duration: 200 }
                                        running: streaming == 1 ? true : false
                                        loops: Animation.Infinite
                                    }      


                                    MouseArea {
                                        anchors.fill: parent
                                        onClicked: {
                                            videoList.currentIndex = index
                                            switchPreview(index,url)
                                        }
                                        onDoubleClicked:{
                                           switchStreaming(index,url)
                                        }
                                
                                    }
                                }
                            }

                            Component{
                                id:addStreamView
                                Rectangle{
                                    height:previewVideoHeight + previewVideoPadding * 2
                                    width:previewVideoHeight * previewVideoRatio + previewVideoPadding * 6
                                    color: "transparent"      
                                    Rectangle {
                                        height:parent.height
                                        width:parent.width - previewVideoPadding * 4
                                        anchors.right:parent.right
                                        color: videoInactiveBgColor
                                        radius:universalBorderRadius
                                        border{
                                            color: videoInactiveBorderColor
                                            width: previewVideoBorderWidth
                                        }   
                                        Text{
                                            anchors.centerIn:parent
                                            color: videoInactiveBorderColor
                                            text: "+"
                                            clip:true
                                            font{
                                                pointSize: parent.height * sensorTextParentHeightPercent
                                            }
                                        }
                                    }    
                                    MouseArea{
                                        anchors.fill: parent
                                        onClicked:{
                                            newStreamPopUp.open()
                                        }
                                    }                                                                    
                                }                           
                            }

                            ListView {
                                id:videoList
                                anchors.fill: parent
                                orientation:ListView.Horizontal
                                spacing:previewBarPadding
                                currentIndex:0
                                model: streamModel
                                delegate: streamingDelegate
                                highlight: Rectangle { color:videoPreviewingBorderColor;border.color: videoPreviewingBorderColor;border.width: previewVideoBorderWidth;radius:universalBorderRadius}
                                focus: true
                                highlightFollowsCurrentItem:true
                                highlightMoveDuration: 200
                                footer:addStreamView 
                                Component.onCompleted: {
                                    mainvideo.command(["loadfile", streamModel.get(videoList.currentIndex).url])    
                                    setMainVideoPlayerVolume(defaultMainVideoVolume)  
                                    mainvideo.setProperty("load-stats-overlay","yes")
                                               
                                }
                            }

                        }

                    }
                    Component.onCompleted: {
                        showPreviews()
                    }
                }                
            }

        }

        Popup {
            Material.theme: Material.Light
            Material.elevation: 6
            id: newStreamPopUp
            modal: true
            focus: true
            closePolicy:Popup.CloseOnPressOutside
            dim:true
            x: Screen.width / 3
            y: Screen.height / 3
            width: Screen.width / 3 
            height:Screen.height / 6
            Row{
                id:newStreamPopContent
                spacing: 20
                anchors.fill:parent
                TextField {
                    id: streamAddField
                    placeholderText: "拉流地址"
                    width:(parent.width - 20) * 0.7
                    anchors.verticalCenter:parent.verticalCenter
                }
                Button {
                    id: button
                    text: "确认"
                    width: (parent.width - 20) * 0.3
                    highlighted: true
                    anchors.verticalCenter:parent.verticalCenter
                }
            }
        }

        Popup {
            Material.theme: Material.Light
            Material.elevation: 6
            id: loadingPopUp
            modal: true
            focus: false
            closePolicy:Popup.NoAutoClose
            dim:true
            x: (Screen.width - loadingIndicatorSize * 2) / 2
            y: (Screen.height - loadingIndicatorSize * 2) / 2
            width: loadingIndicatorSize * 2 
            height:loadingIndicatorSize * 2
            BusyIndicator {
                id:loadingIndicator
                width: loadingIndicatorSize
                height: loadingIndicatorSize
                anchors.centerIn:parent
                running:true
            }
        }        
    }
}
