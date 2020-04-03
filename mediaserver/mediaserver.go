package mediaserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tidwall/gjson"
)

var (
	HTTP_PORT                          = "9900"
	RESTFUL_URL                        = "http://127.0.0.1/index/api/"
	MEDIASERVER_DYLD_LIBRARY_PATH      = ""
	MEDIASERVER_BINARY_PATH            = ""
	ON_STREAM_CHANGE_HANDLER_URL       = "/hook/on_stream_changed"
	ON_MEDIASERVER_STARTED_HANDLER_URL = "/hook/on_server_started"
	ON_STREAM_PLAY_HANDLER_URL         = "/hook/on_play"
	ON_STREAM_PUBLISH_HANDLER_URL      = "/hook/on_publish"
	ON_STREAM_NONE_READER_HANDLER_URL  = "/hook/on_stream_none_reader"
	ON_STREAM_NOT_FOUND_HANDLER_URL    = "/hook/on_stream_not_found"
	STREAM_PROXY_APPNAME               = "proxy"
	LOCAL_STREAM_AUTH_URL_KEY          = "sec"
	LOCAL_STREAM_AUTH_URL_PASSWORD     = "12359"
)

type Stream struct {
	Schema   string
	Vhost    string
	AppName  string
	StreamId string
	UID      string
}

type MediaServer struct {
	Streams       map[string]*Stream
	ProxyMap      map[string]string
	EventServer   *http.Server
	Mux           sync.Mutex
	ServerStarted chan bool
	ServerPid     int
}

func NewMediaServer() *MediaServer {

	return &MediaServer{Streams: make(map[string]*Stream), ProxyMap: make(map[string]string), ServerStarted: make(chan bool)}
}

func (s *MediaServer) StartMediaServerDaemon() error {

	os.Setenv("DYLD_LIBRARY_PATH", MEDIASERVER_DYLD_LIBRARY_PATH)
	cmd := exec.Command(MEDIASERVER_BINARY_PATH, []string{"-d", "&"}...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("\n media server daemon start failed ：%s \n", err)
		return err
	}
	s.ServerPid = cmd.Process.Pid
	return nil
}

func (s *MediaServer) RestartMediaServer() error {

	_, err := http.Get(RESTFUL_URL + "restartServer")
	if err != nil {
		fmt.Printf("\n restart media server command send failed: %s \n", err)
		return err
	}
	return nil
}

func (s *MediaServer) StopMediaServer() {
	syscall.Kill(-s.ServerPid, syscall.SIGKILL)
}

func (s *MediaServer) GetServerConfigItem(itemk string) string {

	res, err := http.Get(RESTFUL_URL + "getServerConfig")

	if err != nil {
		fmt.Printf("\n get media server config item failed: %s \n", err)
		return ""
	}
	contentjson, err := ioutil.ReadAll(res.Body)

	res.Body.Close()
	if err != nil {
		fmt.Printf("\n get media server config item failed: %s \n", err)
		return ""
	}
	//httpport := gjson.Get(string(contentjson), "data.0.http\\.port")
	return gjson.Get(string(contentjson), "data.0."+itemk).String()
}

func (s *MediaServer) SetServerConfigItems(items map[string]string) (send, changed, code int) {

	var str = ""
	changed = 0
	code = -1
	send = 0

	for k, v := range items {
		str += k + "=" + v + "&"
		send += 1
	}

	if str != "" {

		res, err := http.Get(RESTFUL_URL + "setServerConfig?" + str)

		if err != nil {
			fmt.Printf("\n change media server config item failed: %s \n", err)
			return
		}

		contentjson, err := ioutil.ReadAll(res.Body)
		content := string(contentjson)

		res.Body.Close()
		if err != nil {
			fmt.Printf("\n change media server config item failed: %s \n", err)
			return
		}
		changed = int(gjson.Get(content, "changed").Int())
		code = int(gjson.Get(content, "code").Int())

	}

	return

}

func (s *MediaServer) AddStream(st *Stream) {

	s.Mux.Lock()

	if _, ok := s.Streams[st.UID]; ok {
		fmt.Printf("\n already existed stream: %s , stream total : %d\n", st.UID, len(s.Streams))
	} else {
		s.Streams[st.UID] = st
	}

	s.Mux.Unlock()
}

func (s *MediaServer) RemoveStream(st *Stream) {

	s.Mux.Lock()

	if _, ok := s.Streams[st.UID]; ok {

		delete(s.Streams, st.UID)
		fmt.Printf("\n deleted stream: %s\n", st.UID)

	} else {
		fmt.Printf("\n found no stream: %s to delete \n", st.UID)
	}

	s.Mux.Unlock()

}

func (s *MediaServer) AddStreamProxy(rurl string) bool {

	var rtmp string = "0"
	var rtsp string = "0"

	rand.Seed(time.Now().UnixNano())
	streamID := strconv.Itoa(rand.Intn(100))
	u, err := url.Parse(rurl)
	if err != nil {
		fmt.Printf("\n parsing url err : %s \n", err)
		return false
	}

	if strings.ToLower(u.Scheme) == "rtmp" {
		rtmp = "1"
	}
	if strings.ToLower(u.Scheme) == "rtsp" {
		rtsp = "1"
	}
	if _, ok := s.ProxyMap[rurl]; ok {
		fmt.Printf("\n add stream proxy failed: duplicate proxy url \n")
		return false
	}

	res, err := http.Get(RESTFUL_URL + "addStreamProxy?vhost=__defaultVhost__&app=" + STREAM_PROXY_APPNAME + "&stream=" + streamID + "&enable_rtsp=" + rtsp + "&enable_rtmp=" + rtmp + "&enable_hls=0&enable_mp4=0&url=" + rurl)
	if err != nil {
		return false
	}

	contentjson, err := ioutil.ReadAll(res.Body)

	res.Body.Close()

	if err != nil {
		fmt.Printf("\n add stream proxy failed: %s \n", err)
		return false
	}

	content := string(contentjson)

	fmt.Printf("\n added proxy stream :%s\n", content)

	code := gjson.Get(string(content), "code").Int()

	if code == 0 {

		s.Mux.Lock()
		s.ProxyMap[rurl] = gjson.Get(content, "data.key").String()
		s.Mux.Unlock()

		return true
	}

	return false

}

func (s *MediaServer) RemoveStreamProxy(rurl string) bool {

	proxykey := ""

	s.Mux.Lock()

	if k, ok := s.ProxyMap[rurl]; ok {
		proxykey = k
	}

	s.Mux.Unlock()

	if proxykey == "" {
		fmt.Printf("\n delete stream not found \n")
		return false
	}

	res, err := http.Get(RESTFUL_URL + "delStreamProxy?key=" + proxykey)
	if err != nil {
		fmt.Printf("\n delete stream proxy failed: %s \n", err)
		return false
	}

	contentjson, err := ioutil.ReadAll(res.Body)

	res.Body.Close()
	if err != nil {
		fmt.Printf("\n delete stream proxy failed: %s \n", err)
		return false
	}

	content := string(contentjson)

	success := gjson.Get(string(content), "data.flag").Bool()

	if success {
		s.Mux.Lock()
		delete(s.ProxyMap, rurl)
		s.Mux.Unlock()
	}

	return success

}

func (s *MediaServer) StartEventServer() {

	go func() {
		s.EventServer = &http.Server{
			Addr:    ":" + HTTP_PORT,
			Handler: http.DefaultServeMux,
		}

		http.HandleFunc(ON_STREAM_CHANGE_HANDLER_URL, s.OnStreamChanged)
		http.HandleFunc(ON_MEDIASERVER_STARTED_HANDLER_URL, s.OnServerStarted)
		http.HandleFunc(ON_STREAM_PLAY_HANDLER_URL, s.OnPlay)
		http.HandleFunc(ON_STREAM_PUBLISH_HANDLER_URL, s.OnPublish)
		http.HandleFunc(ON_STREAM_NONE_READER_HANDLER_URL, s.OnStreamNoneReader)
		http.HandleFunc(ON_STREAM_NOT_FOUND_HANDLER_URL, s.OnStreamNotFound)

		log.Fatal(s.EventServer.ListenAndServe())
	}()
}

func (s *MediaServer) StopEventServer() {
	s.EventServer.Close()
}

func (s *MediaServer) OnStreamChanged(w http.ResponseWriter, req *http.Request) {

	contentjson, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	content := string(contentjson)
	fmt.Printf("\n streamchanged: %s \n", content)

	if err != nil {
		return
	}

	stream := &Stream{
		Schema:   gjson.Get(content, "schema").String(),
		Vhost:    gjson.Get(content, "vhost").String(),
		AppName:  gjson.Get(content, "app").String(),
		StreamId: gjson.Get(content, "stream").String(),
		UID:      "",
	}

	stream.UID = stream.Schema + ":" + "//127.0.0.1/" + stream.AppName + "/" + stream.StreamId
	register := gjson.Get(content, "regist").Bool()
	if register {
		s.AddStream(stream)
	} else {
		s.RemoveStream(stream)
	}

}

func (s *MediaServer) OnServerStarted(w http.ResponseWriter, req *http.Request) {

	s.ServerStarted <- true
	fmt.Print("\n media server has started  \n")
	req.Body.Close()

	send, changed, code := s.SetServerConfigItems(map[string]string{
		"general.enableVhost":        "0",
		"general.publishToRtxp":      "0",
		"general.publishToHls":       "0",
		"general.publishToMP4":       "0",
		"general.addMuteAudio":       "0",
		"hook.enable":                "1",
		"hook.on_play":               "http://127.0.0.1:" + HTTP_PORT + ON_STREAM_PLAY_HANDLER_URL,
		"hook.on_publish":            "http://127.0.0.1:" + HTTP_PORT + ON_STREAM_PUBLISH_HANDLER_URL,
		"hook.on_stream_changed":     "http://127.0.0.1:" + HTTP_PORT + ON_STREAM_CHANGE_HANDLER_URL,
		"hook.on_stream_none_reader": "http://127.0.0.1:" + HTTP_PORT + ON_STREAM_NONE_READER_HANDLER_URL,
		"hook.on_stream_not_found":   "http://127.0.0.1:" + HTTP_PORT + ON_STREAM_NOT_FOUND_HANDLER_URL,
		"hook.on_server_started":     "http://127.0.0.1:" + HTTP_PORT + ON_MEDIASERVER_STARTED_HANDLER_URL,
		"hook.on_rtsp_realm":         "",
		"hook.on_rtsp_auth":          "",
	})
	fmt.Printf("\n send:%d , changed: %d, code: %d \n", send, changed, code)

}

func (s *MediaServer) OnPlay(w http.ResponseWriter, req *http.Request) {

	contentjson, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	content := string(contentjson)

	if err != nil {
		return
	}

	response := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{
		-1,
		"failed",
	}

	m, err := url.ParseQuery(gjson.Get(content, "params").String())

	if err != nil {
		fmt.Printf("\n onplay url parsing error : %s \n", err)
		return
	}

	if v, ok := m[LOCAL_STREAM_AUTH_URL_KEY]; ok && strings.Join(v, "") == LOCAL_STREAM_AUTH_URL_PASSWORD {
		response.Code = 0
		response.Msg = "success"
		jsonString, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("\n onplay json encode failed : %s \n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonString)
	}

}

func (s *MediaServer) OnPublish(w http.ResponseWriter, req *http.Request) {

	contentjson, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	content := string(contentjson)

	if err != nil {
		return
	}

	response := struct {
		Code       int    `json:"code"`
		EnableHls  bool   `json:"enableHls"`
		EnableMP4  bool   `json:"enableMP4"`
		EnableRtxp bool   `json:"enableRtxp"`
		Msg        string `json:"msg"`
	}{
		-1,
		false,
		false,
		false,
		"failed",
	}

	m, err := url.ParseQuery(gjson.Get(content, "params").String())

	if err != nil {
		fmt.Printf("\n onpublish url parsing error : %s \n", err)
		return
	}

	if v, ok := m[LOCAL_STREAM_AUTH_URL_KEY]; ok && strings.Join(v, "") == LOCAL_STREAM_AUTH_URL_PASSWORD {
		response.Code = 0
		response.Msg = "success"
		jsonString, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("\n onpublish json encode failed : %s \n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonString)

	}

}

func (s *MediaServer) OnStreamNoneReader(w http.ResponseWriter, req *http.Request) {

	req.Body.Close()

	response := struct {
		Code  int  `json:"code"`
		Close bool `json:"close"`
	}{
		0,
		false,
	}

	jsonString, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("\n on stream none reader json encode failed : %s \n", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonString)

}

func (s *MediaServer) OnStreamNotFound(w http.ResponseWriter, req *http.Request) {

	contentjson, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	if err != nil {
		return
	}

	content := string(contentjson)
	fmt.Printf("\n onstream not found : %s \n", content)
}
