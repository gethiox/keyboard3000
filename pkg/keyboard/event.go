package keyboard

import (
	"os"
	"fmt"
)

type KeyEvent struct {
	device   *inputDevice // event source identifier
	code     uint8
	released bool
}

func (ke KeyEvent) String() string {
	return fmt.Sprintf("[RawEvent, code: 0x%02x, released: %5t, device: \"%s\"]", ke.code, ke.released, ke.device.Name)
}

func NewEvent(device *inputDevice, code uint8, released bool) KeyEvent {
	return KeyEvent{device, code, released}
}

type Handler struct {
	Device inputDevice
	Fd     *os.File
}

func NewHandler(fd *os.File, device inputDevice) Handler {
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

		//      ¯\_(ツ)_/¯                released             pressed
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

func (h Handler) ReadKeys(events chan KeyEvent) {
	for {
		event, err := h.ReadKey()
		if err != nil {
			panic(err)
		}
		events <- event
	}
}
