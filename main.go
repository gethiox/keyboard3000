package main

import (
	"fmt"
	"time"
	"keyboard3000/device"
	"os"
)

func main() {
	var now time.Time

	now = time.Now()
	devices, err := device.ReadDevices()
	fmt.Printf("finding keyboard devices takes me: %s\n", time.Since(now))

	if err != nil {
		panic(err)
	}

	now = time.Now()
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()
		fmt.Printf("%s\n", eventPath)
	}
	fmt.Printf("finding event paths takes me: %s\n", time.Since(now))

	eventsChan := make(chan device.KeyEvent, len(devices)*6)

	for _, dev := range devices {
		eventPath, _ := dev.EventPath()

		fd, err := os.Open(eventPath)
		if err != nil {
			panic(err)
		}
		handler := device.Handler{Fd: fd, Device: dev}

		go handler.ReadKeys(eventsChan)
	}

	for {
		event := <-eventsChan
		//now := time.Now()
		fmt.Printf("code: 0x%02x %3d, released: %5t, device: \"%s\"\n", event.Code, event.Code, event.Released, event.Device.Name)
	}
}
