package mediaserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

const (
	HTTP_PORT                          = "9900"
	RESTFUL_URL                        = "http://127.0.0.1/index/api/"
	MEDIASERVER_DYLD_LIBRARY_PATH      = "/Users/s1ngular/GoWork/src/github.com/organicio/mediaserver/"
	MEDIASERVER_BINARY_PATH            = "/Users/s1ngular/GoWork/src/github.com/organicio/mediaserver/MediaServer"
	ON_STREAM_CHANGE_HANDLER_URL       = "/hook/on_stream_changed"
	ON_MEDIASERVER_STARTED_HANDLER_URL = "/hook/on_server_started"
)

type Stream struct {
	Schema   string
	Vhost    string
	AppName  string
	StreamId string
	UID      string
}

type MediaServer struct {
	Streams map[string]*Stream
	mux     sync.Mutex
}

func (s *MediaServer) StartMediaServerDaemon() error {

	go s.StartEventServer()
	os.Setenv("DYLD_LIBRARY_PATH", MEDIASERVER_DYLD_LIBRARY_PATH)
	cmd := exec.Command(MEDIASERVER_BINARY_PATH, []string{"-d", "&"}...)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("media server daemon start failed ：%s \n", err)
		return err
	}
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

	existed := false
	s.mux.Lock()
	for _, v := range s.Streams {
		if v.UID == st.UID {
			existed = true
		}
	}
	if !existed {
		s.Streams[st.UID] = st
		fmt.Printf("\n addded stream: %s , stream total : %d\n", st.UID, len(s.Streams))
	} else {
		fmt.Printf("\n already existed stream: %s , stream total : %d\n", st.UID, len(s.Streams))
	}

	s.mux.Unlock()
}

func (s *MediaServer) RemoveStream(st *Stream) {

	s.mux.Lock()
	for _, v := range s.Streams {
		if v.UID == st.UID {
			delete(s.Streams, st.UID)
			fmt.Printf("\n deleted stream: %s\n", st.UID)
		}
	}
	s.mux.Unlock()

}

func (s *MediaServer) StartEventServer() {

	http.HandleFunc(ON_STREAM_CHANGE_HANDLER_URL, s.OnStreamChanged)
	http.HandleFunc(ON_MEDIASERVER_STARTED_HANDLER_URL, s.OnServerStarted)
	http.ListenAndServe(":"+HTTP_PORT, nil)
}

func (s *MediaServer) OnStreamChanged(w http.ResponseWriter, req *http.Request) {

	contentjson, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	content := string(contentjson)
	fmt.Println(content)

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

	fmt.Print("\n media server has started  \n")
	req.Body.Close()
	time.Sleep(5 * time.Second)
	send, changed, code := s.SetServerConfigItems(map[string]string{
		"general.enableVhost":        "0",
		"general.publishToRtxp":      "0",
		"general.publishToHls":       "0",
		"general.publishToMP4":       "0",
		"hook.enable":                "1",
		"hook.on_stream_none_reader": " ",
	})
	fmt.Printf("send:%d , changed: %d, code: %d", send, changed, code)
}
