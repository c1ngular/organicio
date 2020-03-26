package main

import (
	"fmt"
	"time"

	"github.com/organicio/mediaserver"
	"github.com/organicio/streamer"
)

func main() {

	var err error
	mserver := mediaserver.NewMediaServer()

	mserver.StartEventServer()

	err = mserver.StartMediaServerDaemon()
	if err != nil {
		fmt.Print(err)
	}

	streamer := streamer.NewSreamer()
	streamer.InitRelayServer()
	if err != nil {
		fmt.Print(err)
	}

	time.Sleep(5000 * time.Second)
}
