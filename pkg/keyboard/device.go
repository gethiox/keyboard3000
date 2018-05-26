package keyboard

import (
	"fmt"
	"github.com/xthexder/go-jack"
	"io/ioutil"
	"math/rand"
	"keyboard3000/pkg/hardware"
	"keyboard3000/pkg/logging"
)

const (
	NoteOn  uint8 = 0x90 // Note On Midi data, first four bit, should be mixed with channel bits (last four bits)`
	NoteOff uint8 = 0x80 // note Off Midi data, first four bit, should be mixed with channel bits (last four bits)

	Note    = iota
	Control

	Panic         // ControlEvents targets
	Reset
	OctaveUp
	OctaveDown
	SemitoneUp
	SemitoneDown
	ChannelUp
	ChannelDown
	ProgramUp
	ProgramDown
	OctaveAdd
	OctaveDel
)

type MidiDevice struct {
	Handler *hardware.Handler
	Config  ConfigStruct

	channel     uint8
	semitones   int
	keyMap      keyMap
	pressedKeys pressedKeys

	events   *chan MidiEvent
	MidiPort *jack.Port
}

type pressedKeys map[uint8]uint8
type keyMap map[uint8]keyBind

type MidiEvent struct {
	Port *jack.Port
	Data jack.MidiData
}

type keyBind struct {
	target   uint8
	bindType int
}

func New(handler *hardware.Handler, eventChan *chan MidiEvent) *MidiDevice {
	config, err := FindConfig(handler.Device.Name)
	if err != nil {
		if err == configNotFoundError {
			data, err := ioutil.ReadFile("./maps/default.yml")
			if err != nil {
				panic(err)
			}
			config, err = loadConfig(data)
			logging.Infof(
				"Shiet, configuration is missed for \"%s\" device, but default loaded at least ¯\\_(ツ)_/¯.\n",
				handler.Device.Name,
			)
		} else {
			panic("semi-ultimate shiet occurred")
		}
	}

	keymap := make(keyMap)

	for k, v := range config.Notes {
		keymap[k] = keyBind{target: v, bindType: Note}
	}
	for k, v := range config.Control {
		keymap[k] = keyBind{target: stringToConst[v], bindType: Control}
	}

	return &MidiDevice{
		Handler:     handler,
		Config:      config,
		keyMap:      keymap,
		pressedKeys: make(pressedKeys),
		events:      eventChan,
	}
}

func (d *MidiDevice) Close() {
	midiData := jack.MidiData{
		Time:   0,
		Buffer: []byte{0xb0 | d.channel, 0x7b, 0x00}, // panic,
	}

	*d.events <- MidiEvent{d.MidiPort, midiData}
}

func (d *MidiDevice) ChangeOctave(value int) {
	d.semitones += 12 * value
}

func (d *MidiDevice) ChangeChannel(value int) {
	d.channel += uint8(value)
}

// main function responsible for processing raw hardware events to Midi
func (d *MidiDevice) HandleRawEvent(event hardware.KeyEvent) {
	logging.Infof("%s\n", d)
	logging.Infof("%s\n", event)
	code := event.Code

	bind, ok := d.keyMap[code]
	if !ok {
		logging.Infof("event not in map")
		return
	}

	switch bind.bindType {
	case Note:
		d.handleNote(bind, event)

	case Control:
		d.handleControl(bind, event)
	default:
		panic("The Ultimatest Shiet I've ever seen")
	}
}

func (d *MidiDevice) handleControl(bind keyBind, event hardware.KeyEvent) {
	if event.Released {
		logging.Infof("Control event have no effect on release state\n")
		return
	}

	switch bind.target {
	case OctaveUp:
		d.ChangeOctave(1)
	case OctaveDown:
		d.ChangeOctave(-1)
	case SemitoneUp:
		d.semitones += 1
	case SemitoneDown:
		d.semitones -= 1
	case ChannelUp:
		d.ChangeChannel(1)
	case ChannelDown:
		d.ChangeChannel(-1)
	case Panic:
		midiData := jack.MidiData{
			Time:   0,
			Buffer: []byte{0xb0 | d.channel, 0x7b, 0x00}, // panic,
		}

		*d.events <- MidiEvent{d.MidiPort, midiData}
	}
}

func (d *MidiDevice) handleNote(bind keyBind, event hardware.KeyEvent) {
	var typeAndChannel byte
	var velocity byte
	var midiData jack.MidiData

	if event.Released {
		note, ok := d.pressedKeys[event.Code]
		if !ok {
			logging.Infof("Shiet that should not happened, ignoring that release event")
			return
		}
		typeAndChannel = NoteOff | d.channel
		velocity = 0

		midiData = jack.MidiData{
			Time:   0,
			Buffer: []byte{typeAndChannel, note, velocity},
		}

		delete(d.pressedKeys, event.Code)

	} else {
		note := bind.target + uint8(d.semitones)
		typeAndChannel = NoteOn | d.channel
		velocity = uint8(rand.Intn(63)) + 64

		midiData = jack.MidiData{
			Time:   0,
			Buffer: []byte{typeAndChannel, note, velocity},
		}
		d.pressedKeys[event.Code] = note
	}

	*d.events <- MidiEvent{d.MidiPort, midiData}
}

func (d *MidiDevice) Process() {
	for {
		keyEvent, err := d.Handler.ReadKey()
		if err != nil {
			panic(err)
		}
		d.HandleRawEvent(keyEvent)
	}
}

func (m MidiEvent) String() string {
	return fmt.Sprintf("MidiEvent, time: 0x%02x, data: [0x%02x, 0x%02x, 0x%02x]), port: \"%s\"", m.Data.Time, m.Data.Buffer[0], m.Data.Buffer[1], m.Data.Buffer[2], m.Port.GetName())
}

func (d *MidiDevice) String() string {
	return fmt.Sprintf("MidiDevice [%s], channel: %2d, octaves: %2d (semitones: %2d)", d.Config.Identification.NiceName, d.channel, d.semitones/12, d.semitones)
}
