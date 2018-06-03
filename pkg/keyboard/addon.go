package keyboard

import (
	"os"
	"keyboard3000/pkg/logging"
	"strings"
	"strconv"
	"github.com/xthexder/go-jack"
	"time"
)

func (d *MidiDevice) pitchAddon() {
	fd, err := os.Open("/dev/ttyUSB0")
	if err != nil {
		logging.Infof("Addon shiet, forget about your stupid addon, it just doesn't work")
		logging.Infof(err.Error())
		return
	}
	defer fd.Close()

	var decoded string
	var buffer = make([]byte, 64)

	for {

		amount, _ := fd.Read(buffer)
		for i := 0; i < amount; i++ {
			b := buffer[i]
			if b == 0x0d { // skips CR
				continue
			}

			if b == 0x0a { // NL
				d.handleCompleteData(decoded)
				decoded = ""
				continue // skips newline adding
			}

			decoded += string(b)
		}
	}
}

var global_x float64 = 4096.0
var global_y float64 = 4096.0
var global_z float64 = 4096.0

const divider = 100

var last_sig time.Time
var resetted bool = true

// todo: fix implementation
func (d *MidiDevice) handleCompleteData(data string) {
	values := strings.Split(data, ",")
	if len(values) == 4 { // make sure there is as many values as I expected
		if d.pitchControl && time.Since(last_sig) > 10*time.Millisecond {
			x, _ := strconv.ParseFloat(values[0], 64)
			y, _ := strconv.ParseFloat(values[1], 64)
			z, _ := strconv.ParseFloat(values[2], 64)

			global_x += float64(x / divider)
			global_y += float64(y / divider)
			global_z += float64(z / divider)

			*d.events <- MidiEvent{d.MidiPort, jack.MidiData{0, []byte{MidiPitchControl | d.channel, byte(int(global_z) >> 7 & 0x7f), byte(global_z) & 0x7f}}}

			logging.Infof("x: %8.2f, Y: %8.2f, Z: %8.2f", x, y, z)
			resetted = false
			last_sig = time.Now()
		} else if !d.pitchControl && !resetted {
			*d.events <- MidiEvent{d.MidiPort, jack.MidiData{0, []byte{MidiPitchControl | d.channel, byte(4096 >> 7) & 0x7f, byte(0)}}}
			global_x = 4096.0
			global_y = 4096.0
			global_z = 4096.0
			resetted = true
		}
	}
}