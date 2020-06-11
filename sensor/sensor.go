package sensor

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/organicio/streamer"
)

const (
	HTTP_PORT = "9911"
)

var startTime time.Time

type Sensor struct {
	temp     string
	humidity string
	wind     string
	speed    string
	gps      string
}

type SensorServer struct {
	server *http.Server
}

func NewSensorServer() *SensorServer {
	startTime = time.Now()
	return &SensorServer{}
}

func (s *SensorServer) StartSensorServer() {

	go func() {
		s.server = &http.Server{
			Addr:    ":" + HTTP_PORT,
			Handler: http.DefaultServeMux,
		}

		http.HandleFunc("/sensor/update", s.OnSensorUpdate)
		log.Fatal(s.server.ListenAndServe())
	}()
}

func (s *SensorServer) OnSensorUpdate(w http.ResponseWriter, req *http.Request) {
	rand.Seed(time.Now().UnixNano())
	sensorInfo := &Sensor{
		temp:     strconv.Itoa(rand.Intn(100)),
		humidity: strconv.Itoa(rand.Intn(100)),
		wind:     "西南",
		speed:    strconv.Itoa(rand.Intn(100)),
		gps:      "100.2356,26.8740",
	}
	s.UpdateSensorInfoFile(sensorInfo)
}

func (s *SensorServer) StopSensorServer() {
	s.server.Close()
}

func (s *SensorServer) UpdateSensorInfoFile(sensorInfo *Sensor) {

	duration := Parse(time.Since(startTime)).LimitFirstN(2)

	strInfo := []string{
		"温度：" + sensorInfo.temp + "℃" + "\t ",
		"湿度：" + sensorInfo.humidity + "\\%" + "\t ",
		"风向：" + sensorInfo.wind + "\t ",
		"风速：" + sensorInfo.speed + "m/s" + "\t ",
	}

	if err := WriteFileAtomic(streamer.SENSOR_INFO_TEXT_FILE, []byte("[ 老君山野蓝莓谷 ]"+" \t GPS: "+sensorInfo.gps+" \t 当地时间： %{localtime} \t 运行时长："+duration.String()+"\n"+strings.Join(strInfo[:], "")), 0644); err != nil {
		fmt.Printf("updating sensor info error : %s", err)
	}
}

func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir, name := path.Split(filename)
	f, err := ioutil.TempFile(dir, name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if permErr := os.Chmod(f.Name(), perm); err == nil {
		err = permErr
	}
	if err == nil {
		err = os.Rename(f.Name(), filename)
	}
	// Any err should result in full cleanup.
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
