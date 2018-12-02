package logging

import (
	"fmt"
	"time"
)

var LogMessages = make(chan string, 50)

func Info(message string) {
	now := time.Now()

	message = fmt.Sprintf("%s: %s", now.Format("15:04:05.000000000"), message)

	LogMessages <- message
}

func Infof(format string, vs ...interface{}) {
	now := time.Now()

	format = fmt.Sprintf(format, vs...)
	format = fmt.Sprintf("%s: %s", now.Format("15:04:05.000000000"), format)

	LogMessages <- format
}
