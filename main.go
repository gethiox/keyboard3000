package main

import (
	"fmt"
	"time"
	"keyboard3000/pkg/keyboard"
	"os"
	"github.com/xthexder/go-jack"
)

var devicePorts = make(map[string]*jack.Port, 0)
var Events = make(chan keyboard.KeyEvent, 0)
var Client *jack.Client

//var c int

func process(nframes uint32) int {
	select {
	case <-time.After(1 * time.Nanosecond):
		return 0
	case event := <-Events:
		fmt.Printf("%v\n", event)
		//port := devicePorts[event.inputDevice.Name]
		//buffer := port.MidiClearBuffer(nframes)
		//
		//
		//var midiEvent jack.MidiData
		//
		//midiEvent.Time = 0
		//midiEvent.Buffer = []byte{byte((8+c) << 4), 64, 127} // note off/on variable
		//c += 1
		//if c == 2 {
		//	c = 0
		//}
		//
		////midiEvent := jack.MidiData{Buffer: []byte{9 << 4, 64, 127}, Time: 0}
		//fmt.Printf("%v\n", midiEvent)
		//port.MidiEventWrite(&midiEvent, buffer)
	}

	return 0
}

func shutdown() {
	// todo: release pressed keys before client close
	Client.Close()
	os.Exit(0)
}

func main() {
	// collecting input devices
	now := time.Now()
	devices, err := keyboard.ReadDevices()
	fmt.Printf("finding keyboard devices takes me: %s\n", time.Since(now))
	if err != nil {
		panic(err)
	}

	// prints event paths of listed devices
	now = time.Now()
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()
		fmt.Printf("%s\n", eventPath)
	}
	fmt.Printf("finding event paths takes me: %s\n", time.Since(now))

	// opening JackClient
	var status int
	Client, status = jack.ClientOpen("Keyboard3000", jack.NoStartServer)
	if status != 0 {
		panic("jack-Shiet")
	}
	defer Client.Close()
	Client.OnShutdown(shutdown)

	// setting Jack's process callback
	status = Client.SetProcessCallback(process)
	if status != 0 {
		panic("jack-ultimate-shiet")
	}

	eventsChan := make(chan keyboard.KeyEvent, len(devices)*6)
	//var devicePorts map[string]*jack.Port

	// creating device handlers
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()

		fd, err := os.Open(eventPath)
		if err != nil {
			panic(err)
		}
		handler := keyboard.NewHandler(fd, dev)

		fmt.Printf("Run keyboard: \"%s\"\n", dev.Name)

		devicePorts[dev.Name] = Client.PortRegister(dev.Name, jack.DEFAULT_MIDI_TYPE, jack.PortIsOutput, 0)

		go handler.ReadKeys(eventsChan)
	}

	if code := Client.Activate(); code != 0 {
		fmt.Println("Failed to activate client: ", jack.Strerror(code))
		return
	}

	for {
		event := <-eventsChan
		//fmt.Printf("code: 0x%02x %3d, released: %5t, keyboard: \"%s\"\n", event.Code, event.Code, event.Released, event.inputDevice.Name)
		Events <- event
	}
}
