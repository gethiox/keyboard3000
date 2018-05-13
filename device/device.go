package device

import (
	"fmt"
	"errors"
	"strings"
	"strconv"
	"io/ioutil"
	"os"
)

const devices = "/proc/bus/input/devices"

// keyboard Device data, single values hex-encoded
type Device struct {
	// header, hex values (I: Bus=0003 Vendor=0951 Product=16b7 Version=0111)
	bus     int64
	vendor  int64
	product int64
	version int64

	// always there
	Name     string   // (N: Name="Kingston HyperX Alloy FPS Mechanical Gaming Keyboard")
	handlers []string // (H: Handlers=sysrq kbd event3 mouse0)

	// not useful to me
	//phys     string   // (P: Phys=usb-0000:00:14.0-1/input1)
	//sysfs    string   // (S: Sysfs=/devices/pci0000:00/0000:00:14.0/usb3/3-1/3-1:1.1/0003:0951:16B7.0002/input/input3)
	//uniq     string   // (U: Uniq=)

	// bitmasks
	ev int64 // hex

	// not useful to me
	//prop string   //
	//key  []string // hex? bin?
	// and others optional
}

func (d Device) String() string {
	return fmt.Sprintf("bus: 0x%04x, vendor: 0x%04x, product: 0x%04x, version: 0x%04x, handlers: %v, Name: \"%s\"", d.bus, d.vendor, d.product, d.version, d.handlers, d.Name)
}

func (d Device) Event() (string, error) {
	for _, handler := range d.handlers {
		if len(handler) >= 5 && handler[:5] == "event" {
			return handler, nil
		}
	}
	return "", errors.New("shiet")
}

func (d Device) EventPath() (string, error) {
	event, err := d.Event()
	if err != nil {
		return "", err
	}

	eventPath := fmt.Sprintf("/dev/input/%s", event)

	if _, err := os.Stat(eventPath); os.IsNotExist(err) {
		return "", err
	}

	return eventPath, nil
}

func readValues(record string, dev *Device) {
	switch string(record[0]) {
	case "N": // Name
		dev.Name = string(record[9 : len(record)-1])
	case "I": // identification
		parameters := strings.Split(string(record[3:]), " ")

		dev.bus, _ = strconv.ParseInt(string(parameters[0][4:]), 16, 64)
		dev.vendor, _ = strconv.ParseInt(string(parameters[1][7:]), 16, 64)
		dev.product, _ = strconv.ParseInt(string(parameters[2][8:]), 16, 64)
		dev.version, _ = strconv.ParseInt(string(parameters[3][8:]), 16, 64)

	case "H": // handlers
		var handlers []string

		for _, handler := range strings.Split(string(record[12:]), " ") {
			if handler != "" { // space exist after every handler, this handle that
				handlers = append(handlers, handler)
			}
		}

		dev.handlers = handlers

	case "B": // bitmasks
		switch string(strings.Split(string(record[3:]), "=")[0]) {
		case "EV":
			dev.ev, _ = strconv.ParseInt(string(record[6:]), 16, 64)
		}
	}
}

func createRecords(data string) []string {
	return strings.Split(data, "\n")
}

func createSections(data []byte) []string {
	sections := strings.Split(string(data), "\n\n")
	return sections[:len(sections)-1] // because of additional "\n\n" at the end of the file
}

func ReadDevices() ([]Device, error) {
	data, err := ioutil.ReadFile(devices)
	if err != nil {
		return nil, err
	}

	var devices []Device

	for _, section := range createSections(data) {
		dev := Device{}
		for _, record := range createRecords(section) {
			readValues(record, &dev)
		}
		if dev.ev == 0x120013 { // magic ¯\_(ツ)_/¯
			devices = append(devices, dev)
		}
	}

	return devices, nil
}
