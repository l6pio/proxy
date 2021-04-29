package logs

import (
	"l6p.io/proxy/pkg/cfg"
	"l6p.io/proxy/pkg/sys"
	"log"
)

const LogChannelBufferSize = 10000

var LogChannels []chan *LogData

type Logger interface {
	Log(*LogData) error
	Flush()
}

type LogData struct {
	Name      string        `json:"name"`
	Timestamp int64         `json:"timestamp"`
	Url       string        `json:"url"`
	Method    string        `json:"method"`
	Status    int           `json:"status"`
	Trace     *LogTraceData `json:"trace"`
}

type LogTraceData struct {
	DNSDelta           int64 `json:"dnsDelta"`
	DialDelta          int64 `json:"dialDelta"`
	TLSHandshakeDelta  int64 `json:"tlsHandshakeDelta"`
	ConnectDelta       int64 `json:"connectDelta"`
	FirstResponseDelta int64 `json:"firstResponseDelta"`
}

func InitLoggers(config *cfg.Config) {
	LogChannels = append(LogChannels, NewLogChannel("stdout", NewStdoutLog()))

	if config.KafkaAddr != "" && config.KafkaTopic != "" {
		LogChannels = append(LogChannels, NewLogChannel("kafka", NewKafkaLog(config)))
	}
}

func Log(data *LogData) {
	for _, channel := range LogChannels {
		channel <- data
	}
}

func NewLogChannel(name string, logger Logger) chan *LogData {
	channel := make(chan *LogData, LogChannelBufferSize)

	go func() {
		for {
			err := logger.Log(<-channel)
			if err != nil {
				log.Printf("log error: %s", err.Error())
			}
		}
	}()

	sys.AddExitAction(func() {
		log.Printf("Flushing log channel %s ...", name)
		sys.WaitUntilTimeout(cfg.FlushLogBufferTimeout, func() bool {
			return len(channel) == 0
		})
		log.Printf("Flushing log channel %s done", name)
		logger.Flush()
	})
	return channel
}
