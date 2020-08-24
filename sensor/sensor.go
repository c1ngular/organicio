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
	"time"

	"github.com/organicio/streamer"
)

const (
	HTTP_PORT = "9911"
)

var (
	LOCATION_NAME = ""
	GPS           = ""
)

var startTime time.Time

type Sensor struct {
	sname  string
	svalue string
	sunit  string
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

	var Sensors []Sensor
	rand.Seed(time.Now().UnixNano())
	Sensors = append(Sensors, Sensor{sname: "温度", svalue: strconv.Itoa(rand.Intn(100)), sunit: "℃"})
	Sensors = append(Sensors, Sensor{sname: "湿度", svalue: strconv.Itoa(rand.Intn(100)), sunit: "\\%"})
	Sensors = append(Sensors, Sensor{sname: "风向", svalue: "西南", sunit: ""})
	Sensors = append(Sensors, Sensor{sname: "风速", svalue: strconv.Itoa(rand.Intn(100)), sunit: "m/s"})
	s.UpdateSensorInfoFile(Sensors)
}

func (s *SensorServer) StopSensorServer() {
	s.server.Close()
}

func (s *SensorServer) UpdateSensorInfoFile(sensorInfo []Sensor) {

	duration := Parse(time.Since(startTime)).LimitFirstN(2)
	strInfo := "[" + LOCATION_NAME + "] \t " + "GPS：" + GPS + " \t" + "当地时间： %{localtime} \t " + "运行时长：" + duration.String() + "\n" + "实时数据/分钟： \t "
	for _, s := range sensorInfo {
		strInfo += s.sname + "：" + s.svalue + s.sunit + " \t "
	}

	if err := WriteFileAtomic(streamer.SENSOR_INFO_TEXT_FILE, []byte(strInfo), 0644); err != nil {
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
