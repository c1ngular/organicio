package sensor

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/organicio/streamer"
)

const (
	HTTP_PORT = "9911"
)

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

	strInfo := []string{
		"温度：" + sensorInfo.temp + "℃",
		"湿度：" + sensorInfo.humidity + "\\%",
		"风向：" + sensorInfo.wind,
		"风速：" + sensorInfo.speed + "m/s",
		"位置：" + sensorInfo.gps,
	}

	if err := ioutil.WriteFile(streamer.SENSOR_INFO_TEXT_FILE, []byte(strings.Join(strInfo[:], " \t")+"\n"+strings.Join(strInfo[:], " \t")), 0644); err != nil {
		fmt.Printf("updating sensor info error : %s", err)
	}
}
