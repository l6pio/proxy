package logs

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"l6p.io/proxy/pkg/cfg"
	"l6p.io/proxy/pkg/sys"
	"log"
)

type KafkaLog struct {
	Writer *kafka.Writer
}

func NewKafkaLog(config *cfg.Config) *KafkaLog {
	writer := &kafka.Writer{
		Addr:       kafka.TCP(config.KafkaAddr),
		Topic:      config.KafkaTopic,
		Balancer:   &kafka.LeastBytes{},
		Async:      true,
		BatchSize:  100,
		BatchBytes: 10 * 1024 * 1024,
	}
	return &KafkaLog{Writer: writer}
}

func (k *KafkaLog) Log(data *LogData) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return k.Writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(data.Name),
			Value: bytes,
		},
	)
}

func (k *KafkaLog) Flush() {
	log.Print("Flushing Kafka logs ...")
	sys.WaitUntilTimeout(cfg.FlushKafkaLogTimeout, func() bool {
		stats := k.Writer.Stats()
		return stats.Messages == 0 && stats.Writes == 0
	})
	log.Print("Flushing Kafka logs done.")

	log.Print("Closing Kafka writer ...")
	_ = k.Writer.Close()
	log.Print("Closing Kafka writer done.")
}
