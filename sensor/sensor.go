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
	"sync"
	"time"

	"github.com/organicio/streamer"
)

const (
	HTTP_PORT = "9911"
)

var (
	LOCATION_NAME = ""
	GPS           = ""
	StartTime     = time.Now()
)

type Sensor struct {
	sname  string
	svalue string
	sunit  string
}

type Sensors struct {
	Sensors []*Sensor
	Mux     sync.Mutex
}

var Isensors = Sensors{}

type SensorServer struct {
	server *http.Server
}

func NewSensorServer() *SensorServer {
	return &SensorServer{}
}

func (s *SensorServer) StartSensorServer() {
	if streamer.BURN_SENSOR_INFO_TO_VIDEO {
		s.UpdateSensorInfoFile(Isensors.Sensors)
	}

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

	upcomingSensors := []*Sensor{
		{sname: "湿度", svalue: strconv.Itoa(rand.Intn(100)), sunit: "\\%"},
		{sname: "温度", svalue: strconv.Itoa(rand.Intn(100)), sunit: "℃"},
		{sname: "风向", svalue: "西南", sunit: ""},
		{sname: "风速", svalue: strconv.Itoa(rand.Intn(100)), sunit: "m/s"},
	}

	Isensors.Mux.Lock()

	slen := len(Isensors.Sensors)

	if slen > 0 {

		for j := range upcomingSensors {

			notExisted := true

			for i := 0; i < slen; i++ {
				if upcomingSensors[j].sname == Isensors.Sensors[i].sname {
					Isensors.Sensors[i].svalue = upcomingSensors[j].svalue
					notExisted = false
				}
			}

			if notExisted {
				Isensors.Sensors = append(Isensors.Sensors, upcomingSensors[j])
			}

		}

	} else {
		Isensors.Sensors = append(Isensors.Sensors, upcomingSensors...)
	}

	Isensors.Mux.Unlock()

	if streamer.BURN_SENSOR_INFO_TO_VIDEO {
		s.UpdateSensorInfoFile(Isensors.Sensors)
	}

}

func (s *SensorServer) StopSensorServer() {
	s.server.Close()
}

func (s *SensorServer) UpdateSensorInfoFile(sensorInfo []*Sensor) {

	duration := Parse(time.Since(StartTime)).LimitFirstN(3)
	strInfo := "[" + LOCATION_NAME + "] \t " + "GPS：" + GPS + " \t" + "当地时间： %{localtime} \t " + "运行时间：" + duration.String() + "\n" + "实时数据/分钟： \t "
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
