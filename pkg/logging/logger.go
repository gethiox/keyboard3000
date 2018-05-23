package logging

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds) // todo: resolve Lshortfile flag issue

func Infof(message string, v ...interface{}) {
	logger.Printf(message, v)
}
