package device

import (
	"keyboard3000/pkg/keyboard"
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"errors"
)

type MidiDevice struct {
	Handler *keyboard.Handler
	Config  ConfigStruct

	semitones   int
	keyMap      keyMap
	pressedKeys pressedKeys
}

const (
	Note    = iota
	Control = iota
)

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
	Identification Identification `yaml:"identification"`
	Control        map[int]string `yaml:"control"`
	Notes          map[int]uint8  `yaml:"notes"`
	AutoConnect    []string       `yaml:"auto_connect"`
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

func New(handler *keyboard.Handler) *MidiDevice {
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

	return &MidiDevice{
		Handler:     handler,
		Config:      config,
		pressedKeys: *new(pressedKeys),
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
		panic("not in device map")
	}

	switch bind.bindType {
	case Note:
		fmt.Printf("Note, target: %d\n", bind.target)
	case Control:
		fmt.Printf("Control, target: %d\n", bind.target)
	default:
		panic("The Ultimatest Shiet I've ever seen")
	}
	return 0
}
