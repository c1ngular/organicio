package streamer

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	REMOTE_STREAM_AUTH_URL_KEY      = "sec"
	REMOTE_STREAM_AUTH_URL_PASSWORD = "12359"
	STREAMER_PUSH_URL               = ""

	WATERMARK_ENABLED               = false
	WATERMARK_IMG_URL               = ""
	WATERMARK_POSITION_BOTTOM_RIGHT = "overlay=main_w-overlay_w-10:main_h-overlay_h-10"
	WATERMARK_POSITION_BOTTOM_LEFT  = "overlay=10:main_h-overlay_h-10"
	WATERMARK_POSITION_TOP_RIGHT    = "overlay=main_w-overlay_w-10:10"
	WATERMARK_POSITION_TOP_LEFT     = "overlay=10:10"
	WATERMARK_POSITION              = WATERMARK_POSITION_BOTTOM_RIGHT

	MP3S_FOLDER_PATH    = ""
	MP3_LIST_FILENAME   = "mp3s.txt"
	MP3_MERGED_FILENAME = "merged.mp3"
	MP3_BG_ENABLED      = false

	FFMPEG_EXECUTABLE_PATH       = ""
	FFMPEG_TRANSOCDER_BUFFERSIZE = "65535"
	FFMPEG_STREAMER_BUFFERSIZE   = "10000000"
	FFMPEG_STREAMER_FIFO_SIZE    = "100000"

	FFMPEG_STREAM_MAXBITRATE = "1M"
	FFMPEG_STREAM_BUFFERSIZE = "2M"
	FFMPEG_STREAM_FRAMERATE  = "24"

	FFMPEG_STREAM_CRF_LOW    = "34"
	FFMPEG_STREAM_CRF_MEDIUM = "28"
	FFMPEG_STREAM_CRF_HIGH   = "21"

	FFMPEG_VIDEO_CODEC        = "libx264"
	FFMPEG_VIDEO_BITRATE      = "900k"
	FFMPEG_VIDEO_PIXEL_FORMAT = "yuv420p"

	FFMPEG_AUDIO_CODEC         = "aac"
	FFMPEG_AUDIO_BITRATE       = "64k"
	FFMPEG_AUDIO_SAMPLE_FORMAT = "fltp"
	FFMPEG_AUDIO_SAMPLERATE    = "44100"
	FFMPEG_AUDIO_CHANNELS      = "2"

	FFMPEG_MP3_BGSOUND_VOLUME = "0.1"

	RELAYINPORT  int    = 9981
	RELAYOUTPORT int    = 9982
	LOCALHOST    string = "127.0.0.1"
	PACKETSIZE   int    = 1316
)

type Streamer struct {
	currentStreamingUID string
	isStreamingNow      bool
	transCtxCancel      context.CancelFunc
	streamerCtxCancel   context.CancelFunc
	relayConn           *net.UDPConn
	dataBuf             bytes.Buffer
	inCtxCancel         context.CancelFunc
	outCtxCancel        context.CancelFunc
	mux                 sync.Mutex
}

func NewSreamer() *Streamer {

	return &Streamer{}
}

func (s *Streamer) MergeMp3s() {

	var (
		str = ""
	)

	mp3s, err := ioutil.ReadDir(MP3S_FOLDER_PATH)
	if err != nil {
		fmt.Printf("Error reading - %s\n", err)
		return
	}

	for _, mp3 := range mp3s {

		ext := filepath.Ext(mp3.Name())
		if ext != ".mp3" {
			fmt.Printf("skipping %s, ext: '%s'\n", mp3.Name(), ext)
			continue
		}

		if mp3.Name() == MP3_MERGED_FILENAME {
			fmt.Printf("skipping %s, ext: '%s'\n", mp3.Name(), ext)
			continue
		}

		str += "file " + filepath.Join(MP3S_FOLDER_PATH, mp3.Name()) + "\n"

	}

	if str == "" {
		fmt.Printf("\n no  mp3 files found in specified folder \n")
		return
	}

	listContent, _ := ioutil.ReadFile(MP3S_FOLDER_PATH + MP3_LIST_FILENAME)

	if str != string(listContent) {

		err := ioutil.WriteFile(MP3S_FOLDER_PATH+MP3_LIST_FILENAME, []byte(str), 0755)
		if err != nil {
			fmt.Printf("write mp3 list text file failed : %s", err)
			return
		}

		args := []string{
			"-y",
			"-f",
			"concat",
			"-safe", "0",
			"-i", MP3S_FOLDER_PATH + MP3_LIST_FILENAME,
			"-c", "copy",
			MP3S_FOLDER_PATH + MP3_MERGED_FILENAME,
		}
		cmd := exec.Command(FFMPEG_EXECUTABLE_PATH, args...)
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("\n mp3s merge failed %s\n", err)
		}
		fmt.Printf("\n mp3 merge stdOUT and stdError %s\n", stdoutStderr)

		fmt.Println(str)

	} else {
		fmt.Print("\n skipping merge, mp3 files have not changed \n")
	}
}

func (s *Streamer) StartStreamerProcess() {

	args := []string{
		"-re",
		"-i",
		"udp://" + LOCALHOST + ":" + strconv.Itoa(RELAYOUTPORT) + "?buffer_size=" + FFMPEG_STREAMER_BUFFERSIZE + "&fifo_size=" + FFMPEG_STREAMER_FIFO_SIZE + "&overrun_nonfatal=1",
	}

	if _, err := os.Stat(MP3S_FOLDER_PATH + MP3_MERGED_FILENAME); MP3_BG_ENABLED == true && err == nil {
		args = append(args, []string{
			"-stream_loop", "-1",
			"-i", MP3S_FOLDER_PATH + MP3_MERGED_FILENAME,
			"-filter_complex", "[1:a]volume=" + FFMPEG_MP3_BGSOUND_VOLUME + ",apad[A];[0:a][A]amerge[out]",
		}...)
	}

	args = append(args, []string{
		"-c:v", "copy",

		"-c:a", FFMPEG_AUDIO_CODEC,
		"-b:a", FFMPEG_AUDIO_BITRATE,
		"-sample_fmt", FFMPEG_AUDIO_SAMPLE_FORMAT,
		"-ar", FFMPEG_AUDIO_SAMPLERATE,
		"-ac", FFMPEG_AUDIO_CHANNELS,
		"-threads", "2",
		"-strict", "experimental",
	}...)

	if _, err := os.Stat(MP3S_FOLDER_PATH + MP3_MERGED_FILENAME); MP3_BG_ENABLED == true && err == nil {

		args = append(args, []string{
			"-map",
			"0:v",
			"-map",
			"[out]",
		}...)

	}

	u, err := url.Parse(STREAMER_PUSH_URL)
	if err != nil {
		fmt.Printf("parsing remote streaming url faild: %s \n", err)
		return
	}

	if strings.ToLower(u.Scheme) == "rtmp" {
		args = append(args, []string{

			"-y",
			"-r", FFMPEG_STREAM_FRAMERATE,
			//"-flush_packets", "0",
			"-f", "flv",
			//"udp://" + LOCALHOST + ":" + strconv.Itoa(1234) + "?pkt_size=" + strconv.Itoa(PACKETSIZE) + "&buffer_size=" + FFMPEG_TRANSOCDER_BUFFERSIZE + "&overrun_nonfatal=1",
			STREAMER_PUSH_URL + "?" + REMOTE_STREAM_AUTH_URL_KEY + "=" + REMOTE_STREAM_AUTH_URL_PASSWORD,
		}...)
	}
	if strings.ToLower(u.Scheme) == "rtsp" {

		args = append(args, []string{

			"-y",
			"-r", FFMPEG_STREAM_FRAMERATE,
			//"-flush_packets", "0",
			"-f", "rtsp",
			"-rtsp_transport",
			"tcp",
			//"udp://" + LOCALHOST + ":" + strconv.Itoa(1234) + "?pkt_size=" + strconv.Itoa(PACKETSIZE) + "&buffer_size=" + FFMPEG_TRANSOCDER_BUFFERSIZE + "&overrun_nonfatal=1",
			STREAMER_PUSH_URL + "?" + REMOTE_STREAM_AUTH_URL_KEY + "=" + REMOTE_STREAM_AUTH_URL_PASSWORD,
		}...)
	}

	fmt.Printf("%v", args)

	go func() {

		var (
			streamerCtx    context.Context
			stdout, stderr bytes.Buffer
		)

		streamerCtx, s.streamerCtxCancel = context.WithCancel(context.Background())
		cmd := exec.CommandContext(streamerCtx, FFMPEG_EXECUTABLE_PATH, args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Start()
		if err != nil {
			fmt.Printf("streamer start failed ：%s \n", err)
			return
		}

		s.mux.Lock()
		s.isStreamingNow = true
		s.mux.Unlock()

		fmt.Printf("\n streamer started successfully \n")
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("streamer wait error ： %s", err)
		}

		s.mux.Lock()
		s.isStreamingNow = false
		s.dataBuf.Reset()
		s.mux.Unlock()

		fmt.Printf("\n streamer stderr : %s \n", stderr.String())
		fmt.Printf("\n streamer stdout : %s \n", stdout.String())
		fmt.Printf("\n streamer terminated \n")

	}()

}

func (s *Streamer) StopStreamerProcess() {
	s.streamerCtxCancel()
}

func (s *Streamer) StartTranscoderProcess(murl string, crf string, watermarkPos string, vBitrate string, aBitrate string, maxBitrate string, bufsize string) {

	s.MergeMp3s()

	args := []string{
		"-re",
		"-i", murl,
	}

	if _, err := os.Stat(WATERMARK_IMG_URL); watermarkPos != "" && WATERMARK_ENABLED == true && err == nil {
		args = append(args, []string{
			"-i", WATERMARK_IMG_URL,
			"-filter_complex", watermarkPos,
		}...)
	}

	args = append(args, []string{
		"-c:v", FFMPEG_VIDEO_CODEC,
		"-pix_fmt", FFMPEG_VIDEO_PIXEL_FORMAT,
		"-b:v", vBitrate,
		"-c:a", FFMPEG_AUDIO_CODEC,
		"-b:a", aBitrate,
		"-sample_fmt", FFMPEG_AUDIO_SAMPLE_FORMAT,
		"-ar", FFMPEG_AUDIO_SAMPLERATE,
		"-ac", FFMPEG_AUDIO_CHANNELS,

		"-crf", crf,
		"-threads", "2",
		"-strict", "experimental",
		"-maxrate", maxBitrate,
		"-bufsize", bufsize,
		"-r", FFMPEG_STREAM_FRAMERATE,
		"-pass", "1",
		"-flush_packets", "0",
		"-f", "mpegts",
		"udp://" + LOCALHOST + ":" + strconv.Itoa(RELAYINPORT) + "?pkt_size=" + strconv.Itoa(PACKETSIZE) + "&buffer_size=" + FFMPEG_TRANSOCDER_BUFFERSIZE + "&overrun_nonfatal=1",
	}...)

	go func() {

		var (
			transCtx       context.Context
			stdout, stderr bytes.Buffer
		)

		transCtx, s.transCtxCancel = context.WithCancel(context.Background())
		cmd := exec.CommandContext(transCtx, FFMPEG_EXECUTABLE_PATH, args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Start()
		if err != nil {
			fmt.Printf("\n transcoder start failed ：%s \n", err)
			return
		}
		s.mux.Lock()
		s.currentStreamingUID = murl
		s.mux.Unlock()

		fmt.Printf("\n transcoder started successfully \n")
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("\n transcoder wait error ： %s", err)
		}
		s.mux.Lock()
		s.currentStreamingUID = ""
		s.dataBuf.Reset()
		s.mux.Unlock()

		fmt.Printf("\n transcoder stderr : %s \n", stderr.String())
		fmt.Printf("\n transcoder stdout : %s \n", stdout.String())
		fmt.Printf("\n transcoder terminated \n")

	}()

}

func (s *Streamer) StopTranscoderProcess() {

	s.transCtxCancel()
}

func (s *Streamer) InitRelayServer() error {

	var err error
	s.relayConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(LOCALHOST), Port: RELAYINPORT})
	if err != nil {
		fmt.Println(err)
		return err
	}

	var (
		inctx  context.Context
		outctx context.Context
	)

	inctx, s.inCtxCancel = context.WithCancel(context.Background())
	outctx, s.outCtxCancel = context.WithCancel(context.Background())

	go func(inctx context.Context) {

		for {

			indata := make([]byte, PACKETSIZE)
			insize, _, err := s.relayConn.ReadFromUDP(indata)
			if err != nil {
				fmt.Printf("error during read: %s", err)
			}

			if insize > 0 {

				//fmt.Printf("\n read incoming bytes: %s %d\n", remoteAddr, insize)

				s.mux.Lock()
				_, _ = s.dataBuf.Write(indata)
				s.mux.Unlock()

				//fmt.Printf("\n writing to buffer bytes: %d , total buffer size : %d \n", wbsize, s.dataBuf.Len())
			}

			select {
			case <-inctx.Done():
				fmt.Printf("\n udp reader terminated : %s \n", inctx.Err())
				return
			default:
				continue
			}

		}
	}(inctx)

	go func(outctx context.Context) {

		dst, err := net.ResolveUDPAddr("udp", LOCALHOST+":"+strconv.Itoa(RELAYOUTPORT))
		if err != nil {
			fmt.Printf("udp sender init resolved failed : %s", err)
			return
		}

		for {

			outdata := make([]byte, PACKETSIZE)

			s.mux.Lock()

			if s.dataBuf.Len() > 0 {

				_, err := s.dataBuf.Read(outdata)
				if err != nil {
					fmt.Println(err)
				}

				//fmt.Printf("\n reading buffer bytes: %d\n", rbufsize)

				_, err = s.relayConn.WriteTo(outdata, dst)

				//fmt.Printf("\n sending out bytes: %d\n", outsize)

				if err != nil {
					fmt.Println(err)
				}
			}

			s.mux.Unlock()

			select {
			case <-outctx.Done():
				fmt.Printf("\n udp sender terminated : %s \n", outctx.Err())
				return
			default:
				continue
			}

		}

	}(outctx)

	return err

}

func (s *Streamer) StopRelayServer() {
	s.outCtxCancel()
	s.inCtxCancel()
	s.relayConn.Close()
}
