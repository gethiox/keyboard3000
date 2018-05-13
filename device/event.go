package device

import (
	"os"
)

type Handler struct {
	Device Device
	Fd *os.File
}

type KeyEvent struct {
	Device   Device
	Code     uint8
	Released bool
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

	return KeyEvent{h.Device, event[2], released}, nil
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