package mq

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
	"time"
)

var (
	ErrEmptyBrokers       = errors.New("empty brokers")
	ErrEmptyTopics        = errors.New("empty topics")
	ErrEmptyTopicName     = errors.New("empty topic name")
	ErrUnsupportedPayload = errors.New("unsupported payload")
)

type Message struct {
	Payload Payload     `json:"payload,omitempty"`
	Key     string      `json:"key,omitempty"`
	Body    interface{} `json:"body,omitempty"`
}

func (msg *Message) ParseBody(dst interface{}) error {
	b, err := json.Marshal(msg.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

type Producer struct {
	saramaProducer sarama.AsyncProducer
	topics         map[uint32]string
}

type ProducerConfig struct {
	Brokers []string          `json:"brokers,omitempty"`
	Topics  map[uint32]string `json:"topics,omitempty"`
}

func (c *ProducerConfig) validate() error {
	if len(c.Brokers) == 0 {
		return ErrEmptyBrokers
	}

	if len(c.Topics) == 0 {
		return ErrEmptyTopics
	}

	for task, topic := range c.Topics {
		if topic == "" {
			return ErrEmptyTopicName
		}

		if _, ok := Payloads[Payload(task)]; !ok {
			return ErrUnsupportedPayload
		}
	}

	return nil
}

func NewProducer(ctx context.Context, cfg ProducerConfig) (*Producer, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Flush.Frequency = 500 * time.Millisecond
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	saramaConfig.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}

	go func() {
		for err := range producer.Errors() {
			log.Ctx(ctx).Error().Msgf("sarama produce error, topic: %s, value: %v, err: %v",
				err.Msg.Topic, err.Msg.Value, err.Err.Error())
		}
	}()

	return &Producer{
		saramaProducer: producer,
		topics:         cfg.Topics,
	}, nil
}

func (p *Producer) Close() error {
	return p.saramaProducer.Close()
}

func (p *Producer) SendMessage(msg *Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	saramaMsg := &sarama.ProducerMessage{
		Topic: p.topics[uint32(msg.Payload)],
		Key:   sarama.StringEncoder(msg.Key),
		Value: sarama.ByteEncoder(b),
	}
	p.saramaProducer.Input() <- saramaMsg

	return nil
}
