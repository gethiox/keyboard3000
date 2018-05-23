package main

import (
	"fmt"
	"github.com/xthexder/go-jack"
	"keyboard3000/pkg/hardware"
	"keyboard3000/pkg/keyboard"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	devicePorts = make(map[string]*jack.Port)   // just an local unused collection of opened midi ports
	MidiEvents  = make(chan keyboard.MidiEvent) // main midi event channel
	Client      *jack.Client                    // global Jack client
)

// midi event processing callback
func process(nframes uint32) int {
	select {
	case <-time.After(10 * time.Microsecond):
		return 0
	case event := <-MidiEvents:
		fmt.Printf("%s\n", event)
		buffer := event.Port.MidiClearBuffer(nframes)
		event.Port.MidiEventWrite(&event.Data, buffer)
	}

	return 0
}

func shutdown() {
	// todo: release pressed keys before client close
	Client.Close()
	fmt.Printf("App shut down\n")
	os.Exit(0)
}

func attachSigHandler() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		switch <-sigc {
		default:
			shutdown()
		}
	}()
}

// plox JACK server for keyboard socket
func midiSocketPlox(name string) *jack.Port {
	port := Client.PortRegister(name, jack.DEFAULT_MIDI_TYPE, jack.PortIsOutput, 0)
	if port != nil {
		return port
	}

	for i := 0; i < 128; i++ {
		portName := fmt.Sprintf("%s_%d", name, i)
		port := Client.PortRegister(portName, jack.DEFAULT_MIDI_TYPE, jack.PortIsOutput, 0)
		if port != nil {
			return port
		}
	}
	panic("port-related shiet occurred")
}

func main() {
	attachSigHandler()

	// collecting input devices
	now := time.Now()
	devices, err := hardware.ReadDevices()
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

	// creating device handlers
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()

		fd, err := os.Open(eventPath)
		if err != nil {
			panic(err)
		}
		handler := hardware.NewHandler(fd, dev)
		midiDevice := keyboard.New(&handler, &MidiEvents)
		midiPort := midiSocketPlox(midiDevice.Config.Identification.NiceName)
		midiDevice.MidiPort = midiPort

		event, err := dev.Event()
		if err != nil {
			continue
		}
		devicePorts[event] = midiPort

		fmt.Printf("Run keyboard: \"%s\"\n", dev.Name)
		go midiDevice.Process()
	}

	if code := Client.Activate(); code != 0 {
		fmt.Println("Failed to activate client: ", jack.Strerror(code))
		return
	}

	for {
		time.Sleep(time.Millisecond * 10) // ¯\_(ツ)_/¯
	}
}
