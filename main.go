package main

import (
	"fmt"
	"time"

	"github.com/organicio/mediaserver"
	"github.com/organicio/streamer"
)

func main() {
	mserver := &mediaserver.MediaServer{Streams: make(map[string]*mediaserver.Stream)}
	err := mserver.StartMediaServerDaemon()
	if err != nil {
		fmt.Println(err)
	}

	Streamer := &streamer.Streamer{}
	Streamer.MergeMp3s()

	err = Streamer.InitRelayServer()
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(5 * time.Second)
	fmt.Print(mserver.GetServerConfigItem("hook\\.on_stream_changed"))
	time.Sleep(500 * time.Second)
}
