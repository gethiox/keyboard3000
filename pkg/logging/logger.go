package logging

import (
	"fmt"
	"time"
)

var LogMessages = make(chan string, 10)
//var logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds) // todo: resolve Lshortfile flag issue

func Infof(message string, v ...interface{}) {
	now := time.Now()

	if message[len(message)-1:] == "\n" {
		message = message[0:len(message)-1]
	}

	message = fmt.Sprintf("%s [I] %s", now.Format("15:04:05.000"), message)


	LogMessages <- fmt.Sprintf(message, v...)

	//logger.Printf(message, v...)
}
