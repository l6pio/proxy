package logs

import (
	"encoding/json"
	"log"
)

type StdoutLog struct{}

func NewStdoutLog() *StdoutLog {
	return &StdoutLog{}
}

func (*StdoutLog) Log(data *LogData) error {
	writeToStdout(data)
	return nil
}

func (*StdoutLog) Flush() {
}

func writeToStdout(data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	log.Print(string(dataBytes))
}
