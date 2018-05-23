package logging

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.Ltime | log.Lmicroseconds | log.Lshortfile)

func Infof(message string, v ...interface{}) {
	logger.Printf(message, v)
}