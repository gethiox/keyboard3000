package modifiers

import "github.com/xthexder/go-jack"

type Modifier interface {
	HandleMidi(midi jack.MidiData) []jack.MidiData
}
