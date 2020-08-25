package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/organicio/mediaserver"
	"github.com/organicio/sensor"
	"github.com/organicio/streamer"
	"github.com/tidwall/gjson"
)

var DEVICE_UID = ""
var BUSINESS_UID = ""
var mserver = mediaserver.NewMediaServer()
var mstreamer = streamer.NewStreamer()
var msensor = sensor.NewSensorServer()
var tickerRotate *time.Ticker
var tickerStopSignal = make(chan bool)

func loadConfig(configfilename string) {

	data, err := ioutil.ReadFile(configfilename)
	if err != nil {
		fmt.Printf("ReadFile %s error:%v", configfilename, err)
		panic(err)
	}

	dstr := string(data)
	results := gjson.Parse(dstr)
	deviceuid := results.Get("device_uid").String()
	businessuid := results.Get("business_uid").String()
	locationName := results.Get("location_name").String()
	gps := results.Get("gps").String()
	watermarkEnabled := results.Get("watermark_enabled").Bool()
	watermarkUrl := results.Get("watermark_img_url").String()
	watermarkPosition := results.Get("watermark_position").String()
	mp3BgEnabled := results.Get("mp3_bg_enabled").Bool()
	mp3FolderPath := results.Get("mp3_folder_path").String()
	mp3Volume := results.Get("mp3_bg_volume").String()
	ffmpegPath := results.Get("ffmpeg_binary_path").String()
	stream_push_url := results.Get("stream_push_url").String()
	mediaServerLibPath := results.Get("media_server_lib_path").String()
	mediaServerPath := results.Get("media_server_binary_path").String()
	localAuthKey := results.Get("local_auth_key").String()
	localAuthPass := results.Get("local_auth_pass").String()
	burnSensorInfo := results.Get("burnSensorInfo").Bool()
	starttime := results.Get("statsDatetime").String()
	sensortxtfile := results.Get("sensor_txt_file").String()
	sensortxtfont := results.Get("sensor_txt_fontfile").String()

	if deviceuid == "" || businessuid == "" {
		panic("\n emtpy device uid or business uid \n")
	}

	DEVICE_UID = deviceuid
	BUSINESS_UID = businessuid

	if localAuthKey == "" || localAuthPass == "" {
		panic("\n emtpy auth info uid \n")
	}

	mediaserver.LOCAL_STREAM_AUTH_URL_KEY = localAuthKey
	mediaserver.LOCAL_STREAM_AUTH_URL_PASSWORD = localAuthPass

	if locationName == "" || gps == "" {
		panic("\n emtpy location info \n")
	}

	sensor.LOCATION_NAME = locationName
	sensor.GPS = gps

	if startUnix, err := strconv.ParseInt(starttime, 10, 64); err == nil {
		sensor.StartTime = time.Unix(startUnix, 0)
	}

	if sensortxtfile == "" || sensortxtfont == "" {
		panic("\n sensor info text file or font not assigned \n")
	}

	streamer.SENSOR_INFO_TEXT_FILE = sensortxtfile
	streamer.SENSOR_INFO_FONT_FILE = sensortxtfont

	if _, err := os.Stat(watermarkUrl); err == nil {

		streamer.WATERMARK_ENABLED = watermarkEnabled
		streamer.WATERMARK_IMG_URL = watermarkUrl
		switch watermarkPosition {
		case "top_left":
			streamer.WATERMARK_POSITION = streamer.WATERMARK_POSITION_TOP_LEFT
		case "top_right":
			streamer.WATERMARK_POSITION = streamer.WATERMARK_POSITION_TOP_RIGHT
		case "bottom_left":
			streamer.WATERMARK_POSITION = streamer.WATERMARK_POSITION_BOTTOM_LEFT
		case "bottom_right":
			streamer.WATERMARK_POSITION = streamer.WATERMARK_POSITION_BOTTOM_RIGHT
		default:
			streamer.WATERMARK_POSITION = streamer.WATERMARK_POSITION_BOTTOM_RIGHT
		}

	}

	if _, err := os.Stat(mp3FolderPath); err == nil {
		streamer.MP3_BG_ENABLED = mp3BgEnabled
		streamer.MP3S_FOLDER_PATH = mp3FolderPath
		streamer.FFMPEG_MP3_BGSOUND_VOLUME = mp3Volume
	}

	if _, err := os.Stat(ffmpegPath); err != nil {
		panic("\n ffmpeg binary not found \n")
	}

	streamer.FFMPEG_EXECUTABLE_PATH = ffmpegPath

	if stream_push_url != "" {
		streamer.STREAMER_PUSH_URL = stream_push_url
	}

	if _, err := os.Stat(mediaServerPath); err != nil {
		panic("\n mediaserver binary not found \n")
	}

	mediaserver.MEDIASERVER_BINARY_PATH = mediaServerPath

	if _, err := os.Stat(mediaServerLibPath); err != nil {
		panic("\n mediaserver lid folder not found \n")
	}

	mediaserver.MEDIASERVER_DYLD_LIBRARY_PATH = mediaServerLibPath
	streamer.BURN_SENSOR_INFO_TO_VIDEO = burnSensorInfo

}

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	//runtime.SetBlockProfileRate(1)
	///runtime.SetMutexProfileFraction(5)

	loadConfig("./config.cfg")
	mstreamer.MergeMp3s()
	mserver.StartEventServer()
	msensor.StartSensorServer()

	err := mserver.StartMediaServerDaemon()
	if err != nil {
		fmt.Print(err)
	}

	<-mserver.ServerStarted

	if !mserver.AddStreamProxy("rtmp://hwzbout.yunshicloud.com/mj1170/h6f7wv") {
		fmt.Print("\n rtmp://hwzbout.yunshicloud.com/mj1170/h6f7wv pulled failed \n")
	}
	if !mserver.AddStreamProxy("rtmp://202.69.69.180:443/webcast/bshdlive-pc") {
		fmt.Print("\n rtmp://202.69.69.180:443/webcast/bshdlive-pc pulled failed \n")
	}

	mstreamer.InitRelayServer()
	if err != nil {
		fmt.Print(err)
	}

	mstreamer.StartStreamerProcess()
	startRotateStreaming()

	time.Sleep(360 * time.Second)

	stopRoateStreaming()
	mstreamer.StopStreamerProcess()
	mserver.StopMediaServer()
	mserver.StopEventServer()
	msensor.StopSensorServer()
	mstreamer.StopTranscoderProcess()
	mstreamer.StopRelayServer()

	time.Sleep(4 * time.Second)
}

func getNextStreamingUrl() string {

	url := ""
	fmt.Printf("\n getting next url ... \n")
	mserver.Mux.Lock()
	for k, _ := range mserver.Streams {

		mstreamer.Mux.Lock()
		if k != mstreamer.CurrentStreamingUID && k != streamer.STREAMER_PUSH_URL {
			url = k
			mstreamer.Mux.Unlock()
			break
		}
		mstreamer.Mux.Unlock()

	}
	mserver.Mux.Unlock()

	fmt.Printf("\n current streaming url: %s , next url: %s \n", mstreamer.CurrentStreamingUID, url)
	return url
}

func startRotateStreaming() {

	mstreamer.Mux.Lock()
	if mstreamer.CurrentStreamingUID != "" {
		mstreamer.StopTranscoderProcess()
	}
	mstreamer.Mux.Unlock()

	if url := getNextStreamingUrl(); url != "" {
		mstreamer.StartTranscoderProcess(url, streamer.FFMPEG_STREAM_CRF_LOW, streamer.WATERMARK_POSITION, streamer.FFMPEG_VIDEO_BITRATE, streamer.FFMPEG_AUDIO_BITRATE, streamer.FFMPEG_STREAM_MAXBITRATE, streamer.FFMPEG_STREAM_BUFFERSIZE)

	} else {
		fmt.Printf("\n failed to get next streaming url \n")
	}

	tickerRotate = time.NewTicker(60 * time.Second)

	go func() {

		for {
			select {
			case <-tickerStopSignal:
				return
			case <-tickerRotate.C:

				nextUrl := getNextStreamingUrl()
				if nextUrl != "" {
					if mstreamer.CurrentStreamingUID != "" {
						mstreamer.StopTranscoderProcess()
					}
					mstreamer.StartTranscoderProcess(nextUrl, streamer.FFMPEG_STREAM_CRF_LOW, streamer.WATERMARK_POSITION, streamer.FFMPEG_VIDEO_BITRATE, streamer.FFMPEG_AUDIO_BITRATE, streamer.FFMPEG_STREAM_MAXBITRATE, streamer.FFMPEG_STREAM_BUFFERSIZE)
				} else {
					fmt.Printf("\n failed to get Next rotating stream \n")
				}
			default:
				continue
			}

		}

	}()
}

func stopRoateStreaming() {
	tickerRotate.Stop()
	tickerStopSignal <- true
}
