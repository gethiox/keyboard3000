package hardware

import (
	"fmt"
	"os"
)

type KeyEvent struct {
	device   *InputDevice // event source identifier
	Code     uint8
	Released bool
}

func (ke KeyEvent) String() string {
	return fmt.Sprintf("RawEvent Code: (hex: 0x%02x, decimal: %3d) Released: %-5t device: \"%s\"", ke.Code, ke.Code, ke.Released, ke.device.Name)
}

func NewEvent(device *InputDevice, code uint8, released bool) KeyEvent {
	return KeyEvent{device, code, released}
}

type Handler struct {
	Device InputDevice
	Fd     *os.File
}

func NewHandler(fd *os.File, device InputDevice) Handler {
	return Handler{Fd: fd, Device: device}
}

// kind of reverse engineering because too lazy to understand linux's input.h events structure
func (h Handler) ReadKey() (KeyEvent, error) {
	// at least 24.byte, full 3-part event data
	// first two parts are time-related
	buf := make([]byte, 24)

	var event []byte

	for {
		_, err := h.Fd.Read(buf)
		if err != nil {
			return KeyEvent{}, err
		}

		event = buf[16:23]

		//      ¯\_(ツ)_/¯                Released             pressed
		if event[0] == 0x01 && (event[4] == 0x00 || event[4] == 0x01) {
			break
		}
	}

	var released bool
	if event[4] == 0 {
		released = true
	} else if event[4] == 1 {
		released = false
	} else {
		panic("Ultimate Shiet 6k")
	}

	return NewEvent(&h.Device, event[2], released), nil
}
