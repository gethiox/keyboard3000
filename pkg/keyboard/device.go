package keyboard

import (
	"fmt"
	"github.com/xthexder/go-jack"
	"io/ioutil"
	"keyboard3000/pkg/hardware"
	"keyboard3000/pkg/logging"
	"keyboard3000/pkg/modifiers"
	"math/rand"
)

const (
	//  channel info are in first byte and last four bits
	MidiNoteOn         uint8 = 0x90 // first byte, first four bit mask, should be mixed with channel bits (last four bits)`
	MidiNoteOff        uint8 = 0x80
	MidiControlAndMode uint8 = 0xb0
	MidiProgramChange  uint8 = 0xc0
	MidiPitchControl   uint8 = 0xe0

	MidiPanic uint8 = 0x7b // all notes off (status bytes)

	Note = iota
	Control

	PitchControl
	PitchControlToggle

	Panic  // ControlEvents targets
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

	channel   uint8
	semitones int8
	program   uint8

	keyMap      keyMap
	pressedKeys pressedKeys
	midiPresses midiPresses

	events   *chan MidiEvent
	MidiPort *jack.Port

	modifiers []modifiers.Modifier

	pitchControl bool
}

// oressedKyes keeps track on which keyboard button triggered which midi note
type pressedKeys map[uint8]map[uint8]uint8 // map[Channel][keyboardCode]MidiNote
// midiPresses keeps track on how many times was pressed down
type midiPresses map[uint8]map[uint8]uint8 // map[Channel][MidiNote]times

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
				"Shiet, configuration is missed for \"%s\" device, but default loaded at least ¯\\_(ツ)_/¯.",
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

	device := &MidiDevice{
		Handler:     handler,
		Config:      config,
		keyMap:      keymap,
		pressedKeys: make(pressedKeys),
		midiPresses: make(midiPresses),
		events:      eventChan,
	}

	// generate maps for channels
	for i := 0; i < 16; i++ {
		device.midiPresses[uint8(i)] = make(map[uint8]uint8)
		// generation for pressedKeys is not needed because maps there are generated dynamically for performance purpose (I hope)
	}

	for _, v := range config.Control {
		if v == "pitch_control" {
			go device.pitchAddon()
		}
	}

	return device
}

func (d *MidiDevice) Close() {
	midiData := jack.MidiData{
		Time:   0,
		Buffer: []byte{MidiControlAndMode | d.channel, MidiPanic, 0x00},
	}

	*d.events <- MidiEvent{d.MidiPort, midiData}
}

func (d *MidiDevice) ChangeSemitone(value int) {
	d.semitones += int8(value)
}

func (d *MidiDevice) ChangeOctave(value int) {
	d.semitones += 12 * int8(value)
}

func (d *MidiDevice) ChangeChannel(value int) {
	d.channel = (d.channel + uint8(value)) % 16
}

func (d *MidiDevice) ChangeProgram(value int) {
	d.program += uint8(value)

	midiData := jack.MidiData{
		Time:   0,
		Buffer: []byte{MidiProgramChange | d.channel, d.program, 0x00},
	}

	*d.events <- MidiEvent{d.MidiPort, midiData}
}

// main function responsible for processing raw hardware events to Midi
func (d *MidiDevice) HandleRawEvent(event hardware.KeyEvent) {
	code := event.Code

	bind, ok := d.keyMap[code]
	if !ok {
		logging.Infof("%s config [event not in map]", event)
		return
	} else {
		eventType, ok := d.Config.Control[code]
		if !ok {
			eventType = fmt.Sprintf("midi: %d", d.Config.Notes[code])
		}

		logging.Infof("%s [%s]", event, eventType)
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
	switch bind.target {
	case PitchControl:
		if event.Released {
			d.pitchControl = false
		} else {
			d.pitchControl = true
		}
	}

	if event.Released {
		return
	}

	switch bind.target {
	case OctaveUp:
		d.ChangeOctave(1)
	case OctaveDown:
		d.ChangeOctave(-1)
	case SemitoneUp:
		d.ChangeSemitone(1)
	case SemitoneDown:
		d.ChangeSemitone(-1)
	case ChannelUp:
		d.ChangeChannel(1)
	case ChannelDown:
		d.ChangeChannel(-1)
	case ProgramUp:
		d.ChangeProgram(1)
	case ProgramDown:
		d.ChangeProgram(-1)
	case Panic:
		midiData := jack.MidiData{
			Time:   0,
			Buffer: []byte{MidiControlAndMode | d.channel, MidiPanic, 0x00}, // panic,
		}

		*d.events <- MidiEvent{d.MidiPort, midiData}
	case PitchControlToggle:
		if d.pitchControl {
			d.pitchControl = false
		} else {
			d.pitchControl = true
		}

	}
}

func (d *MidiDevice) handleNote(bind keyBind, event hardware.KeyEvent) {
	var typeAndChannel byte
	var velocity byte
	var midiData jack.MidiData

	if event.Released {
		for channel := range d.pressedKeys { // in fact there should not be more than one iteration in most cases
			note, ok := d.pressedKeys[channel][event.Code]
			if !ok { // keyboard key is not pressed on that channel
				logging.Info("somehow that key wasn't pressed yet on that device (and that channel), skipping NoteOff event generation")
				continue
			}

			times, ok := d.midiPresses[channel][note];
			if !ok {
				panic("lokks like totally shiet, I expect to have direct access to that value every time when I wish to")
			}
			if times > 1 { // key already pressed, skipping NoteOff generation event
				delete(d.pressedKeys[channel], event.Code)
				d.midiPresses[channel][note] -= 1
				continue
			}

			typeAndChannel = MidiNoteOff | channel
			velocity = 0

			midiData = jack.MidiData{
				Time:   0,
				Buffer: []byte{typeAndChannel, note, velocity},
			}
			*d.events <- MidiEvent{d.MidiPort, midiData}

			d.midiPresses[channel][note] = 0
			delete(d.pressedKeys[channel], event.Code)

			// remove current channels map if empty
			if l := len(d.pressedKeys[channel]); l == 0 {
				delete(d.pressedKeys, channel)
			}
		}

	} else {
		if _, ok := d.pressedKeys[d.channel]; !ok { // create map on current channel if not exist
			d.pressedKeys[d.channel] = make(map[uint8]uint8)
		}

		note := bind.target + uint8(d.semitones)

		// uncomment if do not want to generate new NoteOn event on each note duplicate
		//if times, ok := d.midiPresses[d.channel][note]; ok && times > 0 { // skips NoteOn generation when note is already pressed by other keyboard key
		//	d.pressedKeys[d.channel][event.Code] = note
		//	d.midiPresses[d.channel][note] += 1
		//	return
		//}

		typeAndChannel = MidiNoteOn | d.channel
		velocity = uint8(rand.Intn(63)) + 64

		midiData = jack.MidiData{
			Time:   0,
			Buffer: []byte{typeAndChannel, note, velocity},
		}
		*d.events <- MidiEvent{d.MidiPort, midiData}
		d.pressedKeys[d.channel][event.Code] = note

		if _, ok := d.midiPresses[d.channel][note]; !ok {
			d.midiPresses[d.channel][note] = 1
		} else {
			d.midiPresses[d.channel][note] += 1
		}
	}

}

func (d *MidiDevice) Process() {
	for {
		keyEvent, err := d.Handler.ReadKey()
		if err != nil {
			break
		}
		d.HandleRawEvent(keyEvent)
	}
}

func (m MidiEvent) String() string {
	return fmt.Sprintf(
		"MidiEvent, time: 0x%02x, data: [0x%02x, 0x%02x, 0x%02x]), port: \"%s\"",
		m.Data.Time, m.Data.Buffer[0], m.Data.Buffer[1], m.Data.Buffer[2], m.Port.GetName(),
	)
}
func (d *MidiDevice) String() string {
	return fmt.Sprintf(
		"MidiDevice, channel: %2d, program: %2d, octaves: %2d (semitones: %2d), active keys: %d, [%s]",
		d.channel, d.program, d.semitones/12, d.semitones%12, len(d.pressedKeys[d.channel]), d.Config.Identification.NiceName,
	)
}
