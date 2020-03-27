package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/organicio/mediaserver"
	"github.com/organicio/streamer"
	"github.com/tidwall/gjson"
)

var DEVICE_UID = ""
var BUSINESS_UID = ""

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
	watermarkEnabled := results.Get("watermark_enabled").Bool()
	watermarkUrl := results.Get("watermark_img_url").String()
	watermarkPosition := results.Get("watermark_position").String()
	mp3BgEnabled := results.Get("mp3_bg_enabled").Bool()
	mp3FolderPath := results.Get("mp3_folder_path").String()
	ffmpegPath := results.Get("ffmpeg_binary_path").String()
	stream_push_url := results.Get("stream_push_url").String()
	mediaServerLibPath := results.Get("media_server_lib_path").String()
	mediaServerPath := results.Get("media_server_binary_path").String()
	localAuthKey := results.Get("local_auth_key").String()
	localAuthPass := results.Get("local_auth_pass").String()

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
		}

	}

	if _, err := os.Stat(mp3FolderPath); err == nil {
		streamer.MP3_BG_ENABLED = mp3BgEnabled
		streamer.MP3S_FOLDER_PATH = mp3FolderPath
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

}

func main() {

	loadConfig("./config.cfg")

	var err error
	mserver := mediaserver.NewMediaServer()

	mserver.StartEventServer()

	err = mserver.StartMediaServerDaemon()
	if err != nil {
		fmt.Print(err)
	}

	<-mserver.ServerStarted

	mstreamer := streamer.NewSreamer()
	mstreamer.InitRelayServer()
	if err != nil {
		fmt.Print(err)
	}

	mstreamer.StartTranscoderProcess("rtmp://127.0.0.1/live/mobile", streamer.FFMPEG_STREAM_CRF_LOW, streamer.WATERMARK_POSITION, streamer.FFMPEG_VIDEO_BITRATE, streamer.FFMPEG_AUDIO_BITRATE, streamer.FFMPEG_STREAM_MAXBITRATE, streamer.FFMPEG_STREAM_BUFFERSIZE)
	mstreamer.StartStreamerProcess()
	time.Sleep(500 * time.Second)
}
