package device

import (
	"keyboard3000/pkg/keyboard"
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"errors"
	"github.com/xthexder/go-jack"
)

type MidiDevice struct {
	Handler *keyboard.Handler
	Config  ConfigStruct

	channel     int
	semitones   int
	keyMap      keyMap
	pressedKeys pressedKeys

	events   *chan MidiEvent
	MidiPort *jack.Port
}

type MidiEvent struct {
	Port *jack.Port
	Data jack.MidiData
}

const (
	NoteOn  = 144
	NoteOff = 128
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

type pressedKeys map[uint8][]uint8
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

func (d MidiDevice) ChangeOctave(octaves int) {
	d.semitones += 12 * octaves
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

func New(handler *keyboard.Handler, eventChan *chan MidiEvent) *MidiDevice {
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
		pressedKeys: *new(pressedKeys),
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
func (d MidiDevice) HandleRawEvent(event keyboard.KeyEvent) int {
	code := event.Code

	bind, ok := d.keyMap[code]
	if !ok {
		//fmt.Printf("not in device map: '%v'\n", code)
		return 1
		//panic("not in device map")
	}

	switch bind.bindType {
	case Note:
		note := bind.target
		var typeAndChannel byte
		var velocity byte

		if event.Released {
			typeAndChannel = NoteOff
			velocity = 0
		} else {
			typeAndChannel = NoteOn
			velocity = 127
		}

		midiData := jack.MidiData{
			0,
			[]byte{typeAndChannel, note, velocity},
		}

		*d.events <- MidiEvent{d.MidiPort, midiData}

		//fmt.Printf("Note, target: %d\n", bind.target)
	case Control:
		//fmt.Printf("Control, target: %d\n", bind.target)
	default:
		//panic("The Ultimatest Shiet I've ever seen")
	}
	return 0
}

func (d MidiDevice) Process() {
	for {
		keyEvent, err := d.Handler.ReadKey()
		if err != nil {
			panic(err)
		}
		d.HandleRawEvent(keyEvent)
	}
}
