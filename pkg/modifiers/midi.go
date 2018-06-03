package modifiers

import "github.com/xthexder/go-jack"

type Modifier interface {
	Run() error
	HandleMidi(midi jack.MidiData) []jack.MidiData
	Close()
}
