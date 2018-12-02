package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/xthexder/go-jack"
	"keyboard3000/pkg/hardware"
	"keyboard3000/pkg/keyboard"
	"keyboard3000/pkg/logging"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

var (
	activeDevices   []hardware.DeviceInfo // active devices
	keyboardDevices = make(map[hardware.InputID]*keyboard.MidiDevice)
	devicePorts     = make(map[hardware.InputID]*jack.Port) // just an local unused collection of opened midi ports
	MidiEvents      = make(chan keyboard.MidiEvent, 6)      // main midi event channel
	Client          *jack.Client                            // global Jack client
)

const appName = "Keyboard3000"
const (
	LogWindow    = "logs"
	DeviceWindow = "devices"
)

// midi event processing callback
func process(nframes uint32) int {
	for _, port := range devicePorts {
		port.MidiClearBuffer(nframes)
	}

	select {
	case event := <-MidiEvents:
		//logging.Infof("%s", event)
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
	logging.Info("App shut down\n")
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
			for ; i < 20; i++ { // trying to open keyboard event device
				if err != nil {
					time.Sleep(time.Millisecond * 50)
					fd, err = os.Open(eventPath)
					continue
				} else {
					logging.Infof("Device event file opened successfully on %d try", i+1)
					break
				}
			}

			if err != nil {
				logging.Infof("Device event failed to open after %d tries", i+1)
				continue
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

			logging.Infof("Run keyboard: \"%s\"", dev.Name)

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

	logging.Infof("finding keyboard devices takes me: %s", time.Since(now))
	if err != nil {
		panic(err)
	}

	// prints event paths of listed devices
	now = time.Now()
	for _, dev := range devices {
		eventPath, _ := dev.EventPath()
		logging.Info(eventPath)
	}
	logging.Infof("finding event paths takes me: %s", time.Since(now))

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
		logging.Infof("Failed to activate client, code: %d", code)
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

	// GUI updaters
	go logUpdate(g)
	go devicesUpdate(g)

	go func() {
		for {
			time.Sleep(time.Millisecond * 20)
			g.Update(layout)

		}

	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		g.Close()
	}
}

func logUpdate(g *gocui.Gui) {
	for {
		v, err := g.View(LogWindow)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		var message string
		for {
			message = <-logging.LogMessages
			_, err := fmt.Fprintf(v, "\n%s", message)
			if err != nil {
				break
			}
		}
	}
}

func devicesUpdate(g *gocui.Gui) {
	for {
		v, err := g.View(DeviceWindow)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// preparing ordering data
		var keys []hardware.InputID
		for inputID := range keyboardDevices {
			keys = append(keys, inputID)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })

		var content []byte
		v.Clear()

		for _, inputID := range keys {
			md := keyboardDevices[inputID]
			content = []byte(md.String() + "\n")
			v.Write(content)

		}

		time.Sleep(time.Millisecond * 20)
	}

}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView(LogWindow, 0, maxY/2, maxX-1, maxY-1); err != nil {
		v.Title = "[Logs]"
		v.Autoscroll = true
		//v.Frame = false
	}

	if v, err := g.SetView(DeviceWindow, 0, 0, maxX-1, maxY/2-1); err != nil {
		v.Title = "[Devices]"
		v.Autoscroll = false
		//v.Frame = false
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	logging.Info("gui quitted")
	return gocui.ErrQuit
}
