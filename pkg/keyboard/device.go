package keyboard

import (
	"errors"
	"fmt"
	"github.com/xthexder/go-jack"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"keyboard3000/pkg/hardware"
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

func (d *MidiDevice) String() string {
	return fmt.Sprintf("MidiDevice [%s], channel: %2d, octaves: %2d (semitones: %2d)", d.Config.Identification.NiceName, d.channel, d.semitones/12, d.semitones)
}

type MidiEvent struct {
	Port *jack.Port
	Data jack.MidiData
}

func (m MidiEvent) String() string {
	return fmt.Sprintf("\"%s\" (time: 0x%02x, data: [0x%02x, 0x%02x, 0x%02x])", m.Port.GetName(), m.Data.Time, m.Data.Buffer[0], m.Data.Buffer[1], m.Data.Buffer[2])
}

const (
	NoteOn  = 0x90
	NoteOff = 0x80
)

const (
	Note    = iota
	Control
)

const (
	Panic        = iota
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

var stringToConst = map[string]uint8{
	"panic":         Panic,
	"reset":         Reset,
	"octave_up":     OctaveUp,
	"octave_down":   OctaveDown,
	"semitone_up":   SemitoneUp,
	"semitone_down": SemitoneDown,
	"channel_up":    ChannelUp,
	"channel_down":  ChannelDown,
	"program_up":    ProgramUp,
	"program_down":  ProgramDown,
	"octave_add":    OctaveAdd,
	"octave_del":    OctaveDel,
}

type keyBind struct {
	target   uint8
	bindType int
}

type pressedKeys map[uint8]uint8
type keyMap map[uint8]keyBind

type Identification struct {
	RealName string `yaml:"real_name"`
	NiceName string `yaml:"nice_name"`
}

// configuration yaml structure
type ConfigStruct struct {
	Identification Identification   `yaml:"identification"`
	Control        map[uint8]string `yaml:"control"`
	Notes          map[uint8]uint8  `yaml:"notes"`
	AutoConnect    []string         `yaml:"auto_connect"`
}

func (d *MidiDevice) ChangeOctave(value int) {
	d.semitones += 12 * value
}

func (d *MidiDevice) ChangeChannel(value int) {
	d.channel += uint8(value)
}

func loadConfig(data []byte) (ConfigStruct, error) {
	var config ConfigStruct //Config := ConfigStruct{}
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return ConfigStruct{}, err
	}
	return config, nil

}

var configNotFoundError = errors.New("shiet, Config not founded")

func New(handler *hardware.Handler, eventChan *chan MidiEvent) *MidiDevice {
	config, err := FindConfig(handler.Device.Name)
	if err != nil {
		if err == configNotFoundError {
			data, err := ioutil.ReadFile("./maps/default.yml")
			if err != nil {
				panic(err)
			}
			config, err = loadConfig(data)
			fmt.Printf(
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

// finds and return KeyMap
func FindConfig(name string) (ConfigStruct, error) {
	files, err := ioutil.ReadDir("./maps/")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile("./maps/" + file.Name())
		if err != nil {
			panic(err)
		}

		config, err := loadConfig(data)
		if err != nil {
			panic(err)
		}

		if name == config.Identification.RealName {
			fmt.Printf("Great, configuration found for \"%s\" device.\n", name)
			return config, nil
		}
	}
	return ConfigStruct{}, configNotFoundError
}

// main function responsible for processing raw hardware events to Midi
func (d *MidiDevice) HandleRawEvent(event hardware.KeyEvent) {
	fmt.Printf("%s\n", d)
	fmt.Printf("%s\n", event)
	code := event.Code

	bind, ok := d.keyMap[code]
	if !ok {
		fmt.Println("event not in map")
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
		fmt.Printf("Control event have no effect on release state\n")
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
			fmt.Println("Shiet that should not happened, ignoring that release event")
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
