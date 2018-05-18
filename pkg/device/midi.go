package device

type pressedKeys map[uint8][]uint8
type keyMap map[uint8]uint8

type midiDevice struct {
	semitones   int
	keyMap      keyMap
	pressedKeys pressedKeys
}

func (d midiDevice) ChangeOctave(octaves int) {
	d.semitones += 12 * octaves
}

func (d midiDevice) New(keyMap keyMap) *midiDevice {
	return &midiDevice{
		semitones:   0,
		keyMap:      keyMap,
		pressedKeys: *new(pressedKeys),
	}
}
