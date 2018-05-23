package hardware

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const devicesPath = "/proc/bus/input/devices"

// keyboard inputDevice data
type inputDevice struct {
	bus     int64
	vendor  int64
	product int64
	version int64

	Name     string
	handlers []string

	ev int64
}

func (d *inputDevice) String() string {
	return fmt.Sprintf("bus: 0x%04x, vendor: 0x%04x, product: 0x%04x, version: 0x%04x, handlers: %v, Name: \"%s\"", d.bus, d.vendor, d.product, d.version, d.handlers, d.Name)
}

// finds event attribute in device handlers array
func (d *inputDevice) Event() (string, error) {
	for _, handler := range d.handlers {
		if len(handler) >= 5 && handler[:5] == "event" {
			return handler, nil
		}
	}
	return "", errors.New("shiet")
}

// returns event file path like /dev/input/event4
func (d *inputDevice) EventPath() (string, error) {
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

// reads parameters from section and update device entity
func readValues(record string, dev *inputDevice) {
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

// reads available keyboard device
func ReadDevices() ([]inputDevice, error) {
	data, err := ioutil.ReadFile(devicesPath)
	if err != nil {
		return nil, err
	}

	var devices []inputDevice

	for _, section := range createSections(data) {
		dev := inputDevice{}
		for _, record := range createRecords(section) {
			readValues(record, &dev)
		}
		if dev.ev == 0x120013 { // magic ¯\_(ツ)_/¯
			devices = append(devices, dev)
		}
	}

	return devices, nil
}
