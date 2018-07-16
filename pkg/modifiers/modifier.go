package modifiers

import (
	"time"
)

const (
	down   = iota
	up
	random
)

type Modifier interface {
	Run() error
	Close() error

}


// Parallel modifier simply add notes in relation to current pressed
//type ParallelModifier struct [
//]



type ArpegioModifier struct {
	//device *keyboard.MidiDevice
	semitoneOffsets []int
	speed float64 // Hz
	direction int

	runned bool
}

func (m *ArpegioModifier) Run() error {
	m.runned = true
	for m.runned {


		time.Sleep(time.Second / 2)
	}
	return nil
}

func (m *ArpegioModifier) Close() error {
	m.runned = false
	return nil
}



