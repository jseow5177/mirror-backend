package mq

import (
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

var (
	ErrInvalidBalanceStrategy = errors.New("invalid balance strategy")
	ErrInvalidInitialOffset   = errors.New("invalid initial offset")
)

type HandlerFunc func(ctx context.Context, msg *Message) error

var (
	handlerLock sync.RWMutex
	handlers    = make(map[Payload]HandlerFunc)
)

func RegisterHandler(payload Payload, handler HandlerFunc) {
	handlerLock.Lock()
	defer handlerLock.Unlock()

	if handler == nil {
		panic("payload handler is nil")
	}

	if _, ok := handlers[payload]; ok {
		panic("payload already has a handler")
	}

	handlers[payload] = handler
}

func getHandlerFunc(payload Payload) HandlerFunc {
	handlerLock.RLock()
	defer handlerLock.RUnlock()
	handler, ok := handlers[payload]
	if !ok {
		return nil
	}
	return handler
}

type ConsumerConfig struct {
	Brokers         []string `json:"brokers,omitempty"`
	Topic           string   `json:"topic,omitempty"`
	ConsumerGroup   string   `json:"consumer_group,omitempty"`
	BalanceStrategy string   `json:"balance_strategy,omitempty"`
	InitialOffset   string   `json:"initial_offset,omitempty"`
}

var balanceStrategies = []string{"sticky", "roundrobin", "range"}

var initialOffsets = []string{"newest", "oldest"}

func (c *ConsumerConfig) validate() error {
	if len(c.Brokers) == 0 {
		return ErrEmptyBrokers
	}

	if c.Topic == "" {
		return ErrEmptyTopicName
	}

	if c.BalanceStrategy != "" && !goutil.ContainsStr(balanceStrategies, c.BalanceStrategy) {
		return ErrInvalidBalanceStrategy
	}

	if c.InitialOffset != "" && !goutil.ContainsStr(initialOffsets, c.InitialOffset) {
		return ErrInvalidInitialOffset
	}

	return nil
}

type Consumer struct {
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	client sarama.ConsumerGroup
	ready  chan bool
}

func NewConsumer(ctx context.Context, cfg ConsumerConfig) (*Consumer, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = true

	if cfg.InitialOffset == "oldest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	switch cfg.BalanceStrategy {
	case balanceStrategies[0]:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategySticky()}
	case balanceStrategies[1]:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	default:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	}

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)

	c := &Consumer{
		ctx:    subCtx,
		client: client,
		cancel: cancel,
		ready:  make(chan bool),
	}

	c.wg = new(sync.WaitGroup)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				if err := client.Consume(c.ctx, []string{cfg.Topic}, c); err != nil {
					if errors.Is(err, sarama.ErrClosedConsumerGroup) {
						return
					}
				}
			}
			c.ready = make(chan bool)
		}
	}()

	<-c.ready

	log.Ctx(c.ctx).Info().Msg("consumer is up and running!")

	return c, nil
}

func (c *Consumer) Close() error {
	c.cancel()
	c.wg.Wait()
	return c.client.Close()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case consumerMessage, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			// init log_id
			ctx := log.With().Str("log_id", uuid.New().String()).Logger().WithContext(c.ctx)

			c.processMessage(ctx, session, consumerMessage)
		case <-session.Context().Done():
			return nil
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, session sarama.ConsumerGroupSession, consumerMessage *sarama.ConsumerMessage) {
	var (
		err   error
		start = time.Now()
		msg   = new(Message)
	)

	defer func() {
		since := time.Now().Sub(start).Microseconds()
		log.Ctx(ctx).Info().Msgf("message processed: value = %s, timestamp = %v, topic = %s, offset = %v, proctm: %vμs, err: %v",
			string(consumerMessage.Value), consumerMessage.Timestamp, consumerMessage.Topic, consumerMessage.Offset, since, err)

		c.markMessage(session, consumerMessage)
	}()

	if err = json.Unmarshal(consumerMessage.Value, msg); err != nil {
		err = fmt.Errorf("failed to unmarshal message: %w", err)
		return
	}

	fn := getHandlerFunc(msg.Payload)
	if fn == nil {
		err = fmt.Errorf("message handler is nil, paylod: %v", msg.Payload)
		return
	}

	if err = fn(ctx, msg); err != nil {
		err = fmt.Errorf("failed to handle message: %w", err)
		return
	}
}

func (c *Consumer) markMessage(session sarama.ConsumerGroupSession, consumerMessage *sarama.ConsumerMessage) {
	session.MarkMessage(consumerMessage, "")
}
