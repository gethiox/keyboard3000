package main

import (
	"fmt"
	"github.com/xthexder/go-jack"
	"keyboard3000/pkg/hardware"
	"keyboard3000/pkg/keyboard"
	"keyboard3000/pkg/logging"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/jroimartin/gocui"
)

var (
	activeDevices   []hardware.DeviceInfo // active devices
	keyboardDevices = make(map[hardware.InputID]*keyboard.MidiDevice)
	devicePorts     = make(map[hardware.InputID]*jack.Port) // just an local unused collection of opened midi ports
	MidiEvents      = make(chan keyboard.MidiEvent, 6)      // main midi event channel
	Client          *jack.Client                            // global Jack client
)

const appName = "Keyboard3000"

// midi event processing callback
func process(nframes uint32) int {
	for _, port := range devicePorts {
		port.MidiClearBuffer(nframes)
	}

	select {
	case event := <-MidiEvents:
		//logging.Infof("%s\n", event)
		buffer := event.Port.MidiClearBuffer(nframes) // todo: port can be cleaned second time here, make sure if that is okay
		event.Port.MidiEventWrite(&event.Data, buffer)
	default:
		return 0
	}

	return 0
}

func shutdown() {
	for _, device := range keyboardDevices {
		device.Close()
	}
	time.Sleep(time.Millisecond * 10) // make sure that Panic events will be processed by jack process() callback
	Client.Close()
	logging.Infof("App shut down\n")
	os.Exit(0)
}

func attachSigHandler() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(
		sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
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

	// in case of already opened port with requested name adding suffixes is tried
	for i := 0; i < 128; i++ {
		portName := fmt.Sprintf("%s_%d", name, i)
		port := Client.PortRegister(portName, jack.DEFAULT_MIDI_TYPE, jack.PortIsOutput, 0)
		if port != nil {
			return port
		}
	}
	panic("port-related shiet occurred")
}

func pluggedDevices(current []hardware.DeviceInfo) []hardware.DeviceInfo {
	var devices []hardware.DeviceInfo

Shiet:
	for _, dev := range current {
		for _, active := range activeDevices {
			if active.Equal(&dev) { // device is already active
				continue Shiet
			}
		}
		devices = append(devices, dev)

	}

	return devices
}

func removedDevices(current []hardware.DeviceInfo) []hardware.DeviceInfo {
	var devices []hardware.DeviceInfo
	var removed bool

	for _, active := range activeDevices {
		removed = true
		for _, dev := range current {
			if active.Equal(&dev) {
				removed = false
				break
			}
		}
		if removed {
			devices = append(devices, active)
		}
	}

	return devices
}

// monitor physical keyboard device connections and create/remove virtual one if needed
func deviceMonitor() {
	// creating device handlers
	for {
		currentDevices, _ := hardware.ReadDevices() // reads current

		for _, dev := range pluggedDevices(currentDevices) {
			eventPath, _ := dev.EventPath()

			fd, err := os.Open(eventPath)
			i := 0
			for ; i < 10; i++ { // trying to open keyboard event device
				if err != nil {
					time.Sleep(time.Millisecond * 50)
					fd, err = os.Open(eventPath)
				} else {
					logging.Infof("Device event file opened successfully on %d try", i+1)
					break
				}
			}
			if err != nil {
				logging.Infof("Device event failed to open after %d tries", i)
				panic(err)
			}

			activeDevices = append(activeDevices, dev) // mark device as active from this point

			handler := hardware.NewHandler(fd, dev)
			midiDevice := keyboard.New(&handler, &MidiEvents)
			midiPort := midiSocketPlox(midiDevice.Config.Identification.NiceName)
			midiDevice.MidiPort = midiPort

			keyboardDevices[dev.Identifier()] = midiDevice
			devicePorts[dev.Identifier()] = midiPort

			for _, target := range midiDevice.Config.AutoConnect {
				targetPort := Client.GetPortByName(target)
				if targetPort != nil {
					code := Client.ConnectPorts(midiPort, targetPort)
					if code != 0 {
						logging.Infof("Autoconnect failed from \"%s\" to \"%s\"", midiPort, targetPort)
					} else {
						logging.Infof("Autoconnect succeeded from \"%s\" to \"%s\"", midiPort, targetPort)
					}
				}
			}

			logging.Infof("Run keyboard: \"%s\"\n", dev.Name)

			go midiDevice.Process()
		}

		toRemoveDevices := removedDevices(currentDevices)
		for _, dev := range toRemoveDevices {
			logging.Infof("remove dev: %v", dev)

			keyboardDev, ok := keyboardDevices[dev.Identifier()]
			if !ok {
				panic("Looks like pre-ultimate shiet occurred")
			}

			keyboardDev.Close()
			Client.PortUnregister(devicePorts[dev.Identifier()])

			delete(keyboardDevices, dev.Identifier())
			delete(devicePorts, dev.Identifier())
			activeDevices = remove(activeDevices, lookupForIndex(activeDevices, dev))
		}

		time.Sleep(time.Millisecond * 200)
	}
}

// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-array-in-golang/37335777
func remove(s []hardware.DeviceInfo, i int) []hardware.DeviceInfo {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func lookupForIndex(slice []hardware.DeviceInfo, value hardware.DeviceInfo) int {
	for i, v := range slice {
		if v.Equal(&value) {
			return i
		}
	}
	return 0
}

func main() {
	attachSigHandler()

	// collecting input devices
	now := time.Now()
	devices, err := hardware.ReadDevices()

	logging.Infof("finding keyboard devices takes me: %s\n", time.Since(now))
	if err != nil {
		panic(err)
	}

	// prints event paths of listed devices
	now = time.Now()
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()
		logging.Infof("%s\n", eventPath)
	}
	logging.Infof("finding event paths takes me: %s\n", time.Since(now))

	// opening JackClient
	var status int
	Client, status = jack.ClientOpen(appName, jack.NoStartServer)
	if status != 0 {
		panic("jack-Shiet")
	}
	defer Client.Close()
	Client.OnShutdown(shutdown)
	defer shutdown()

	// setting Jack's process callback
	status = Client.SetProcessCallback(process)
	if status != 0 {
		panic("jack-ultimate-shiet")
	}

	if code := Client.Activate(); code != 0 {
		logging.Infof("Failed to activate client: ", code)
		return
	}

	go deviceMonitor()
	//
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlQ, gocui.ModNone, quit); err != nil {
		panic(err)
	}

	go func() {
		for {
			time.Sleep(time.Millisecond * 20)
			g.Update(layout)

		}

	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		g.Close()
		//logging.Infof("exited")
		//panic(err)
	}

	//for {
	//	<-logging.LogMessages
	//	time.Sleep(time.Millisecond * 50) // ¯\_(ツ)_/¯s
	//}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	v, err := g.SetView("logs", 0, maxY/2, maxX-1, maxY-1)
	v.Autoscroll = true
	//v.Frame = false

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

ASDF:
	for {
		select {
		case message := <-logging.LogMessages:
			fmt.Fprintf(v, "\n%s", message)
		case <-time.After(time.Millisecond * 10):
			break ASDF
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	logging.Infof("gui quitted")
	return gocui.ErrQuit
}
