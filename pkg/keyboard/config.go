package keyboard

import (
	"gopkg.in/yaml.v2"
	"errors"
	"io/ioutil"
	"keyboard3000/pkg/logging"
)

var configNotFoundError = errors.New("shiet, Config not founded")

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

var stringToConst = map[string]uint8{
	"panic":                Panic,
	"reset":                Reset,
	"octave_up":            OctaveUp,
	"octave_down":          OctaveDown,
	"semitone_up":          SemitoneUp,
	"semitone_down":        SemitoneDown,
	"channel_up":           ChannelUp,
	"channel_down":         ChannelDown,
	"program_up":           ProgramUp,
	"program_down":         ProgramDown,
	"octave_add":           OctaveAdd,
	"octave_del":           OctaveDel,
	"pitch_control":        PitchControl,
	"pitch_control_toggle": PitchControlToggle,
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
			logging.Infof("Great, configuration found for \"%s\" device.\n", name)
			return config, nil
		}
	}
	return ConfigStruct{}, configNotFoundError
}

func loadConfig(data []byte) (ConfigStruct, error) {
	var config ConfigStruct //Config := ConfigStruct{}
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return ConfigStruct{}, err
	}
	return config, nil

}
