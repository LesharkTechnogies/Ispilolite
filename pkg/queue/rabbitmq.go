package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"ispilolite/pkg/monitoring"
)

const (
	JobExchange           = "ispilo.job"
	NotificationExchange  = "ispilo.notification"
	DeadLetterExchange    = "ispilo.deadletter"
	JobAssignmentQueue    = "job.assignment"
	NotificationPushQueue = "notification.push"
	NotificationSMSQueue  = "notification.sms"
)

type Config struct {
	URL          string
	ConsumerName string
	Prefetch     int
}

type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	AggregateID string                 `json:"aggregate_id,omitempty"`
	RecipientID string                 `json:"recipient_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	OccurredAt  time.Time              `json:"occurred_at"`
}

type Handler func(context.Context, Event) error

type Service struct {
	mu            sync.Mutex
	conn          *amqp091.Connection
	pub           *amqp091.Channel
	confirmations <-chan amqp091.Confirmation
	cfg           Config
	consumerName  string
	closed        chan struct{}
	closeOnce     sync.Once
}

var defaultService struct {
	sync.RWMutex
	service *Service
}

func SetDefault(service *Service) {
	defaultService.Lock()
	defaultService.service = service
	defaultService.Unlock()
}

func Publish(ctx context.Context, exchange, key string, event Event) error {
	defaultService.RLock()
	service := defaultService.service
	defaultService.RUnlock()
	if service == nil {
		return errors.New("rabbitmq is not configured")
	}
	return service.Publish(ctx, exchange, key, event)
}

func PublishBestEffort(exchange, key string, event Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = Publish(ctx, exchange, key, event)
}

func New(cfg Config) (*Service, error) {
	if cfg.URL == "" {
		return nil, errors.New("rabbitmq url is required")
	}
	if cfg.Prefetch <= 0 {
		cfg.Prefetch = 20
	}
	s := &Service{cfg: cfg, consumerName: cfg.ConsumerName, closed: make(chan struct{})}
	s.mu.Lock()
	err := s.connectLocked()
	s.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	return s, nil
}

func (s *Service) connectLocked() error {
	conn, err := amqp091.Dial(s.cfg.URL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err = ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	if err = s.declare(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	oldPub, oldConn := s.pub, s.conn
	s.conn = conn
	s.pub = ch
	s.confirmations = ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
	if oldPub != nil {
		_ = oldPub.Close()
	}
	if oldConn != nil {
		_ = oldConn.Close()
	}
	return nil
}

func (s *Service) reconnect(ctx context.Context) error {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closed:
			return errors.New("rabbitmq service is closed")
		default:
		}

		s.mu.Lock()
		if s.conn != nil && !s.conn.IsClosed() && s.pub != nil && !s.pub.IsClosed() {
			s.mu.Unlock()
			return nil
		}
		err := s.connectLocked()
		s.mu.Unlock()
		if err == nil {
			return nil
		}
		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-s.closed:
			timer.Stop()
			return errors.New("rabbitmq service is closed")
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (s *Service) declare(ch *amqp091.Channel) error {
	for _, exchange := range []string{JobExchange, NotificationExchange, DeadLetterExchange} {
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}
	for _, q := range []struct {
		name, exchange, key string
	}{{JobAssignmentQueue, JobExchange, "job.#"}, {NotificationPushQueue, NotificationExchange, "notification.push"}, {NotificationSMSQueue, NotificationExchange, "notification.sms"}} {
		args := amqp091.Table{"x-dead-letter-exchange": DeadLetterExchange, "x-dead-letter-routing-key": q.name + ".dead"}
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, args); err != nil {
			return err
		}
		if err := ch.QueueBind(q.name, q.key, q.exchange, false, nil); err != nil {
			return err
		}
		dlq := q.name + ".dlq"
		if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(dlq, q.name+".dead", DeadLetterExchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Publish(ctx context.Context, exchange, key string, event Event) error {
	tracer := otel.Tracer("ispilolite/queue")
	ctx, span := tracer.Start(ctx, "rabbitmq publish", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	headers := amqp091.Table{}
	for key, value := range carrier {
		headers[key] = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub == nil || s.pub.IsClosed() {
		if err := s.connectLocked(); err != nil {
			monitoring.QueuePublished.WithLabelValues(exchange, key, "error").Inc()
			return err
		}
	}
	err = s.pub.PublishWithContext(ctx, exchange, key, false, false, amqp091.Publishing{DeliveryMode: amqp091.Persistent, ContentType: "application/json", MessageId: event.ID, Type: event.Type, Timestamp: event.OccurredAt, Headers: headers, Body: body})
	if err != nil {
		monitoring.QueuePublished.WithLabelValues(exchange, key, "error").Inc()
		return err
	}
	select {
	case confirmation, ok := <-s.confirmations:
		if !ok || !confirmation.Ack {
			monitoring.QueuePublished.WithLabelValues(exchange, key, "nack").Inc()
			return errors.New("rabbitmq rejected message")
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	monitoring.QueuePublished.WithLabelValues(exchange, key, "success").Inc()
	return nil
}

func (s *Service) Consume(ctx context.Context, queueName string, prefetch int, handler Handler) error {
	if prefetch <= 0 {
		prefetch = s.cfg.Prefetch
	}
	go s.superviseConsumer(ctx, queueName, prefetch, handler)
	return nil
}

func (s *Service) superviseConsumer(ctx context.Context, queueName string, prefetch int, handler Handler) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		ch, deliveries, err := s.startConsumer(ctx, queueName, prefetch)
		if err != nil {
			if s.reconnect(ctx) != nil {
				return
			}
			continue
		}
		backoff = time.Second
		s.consumeDeliveries(ctx, queueName, deliveries, handler)
		_ = ch.Close()
		if ctx.Err() != nil {
			return
		}
		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (s *Service) startConsumer(ctx context.Context, queueName string, prefetch int) (*amqp091.Channel, <-chan amqp091.Delivery, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || conn.IsClosed() {
		return nil, nil, errors.New("rabbitmq connection is closed")
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	if err = ch.Qos(prefetch, 0, false); err != nil {
		_ = ch.Close()
		return nil, nil, err
	}
	deliveries, err := ch.Consume(queueName, s.consumerName, false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, nil, err
	}
	return ch, deliveries, nil
}

func (s *Service) consumeDeliveries(ctx context.Context, queueName string, deliveries <-chan amqp091.Delivery, handler Handler) {
	tracer := otel.Tracer("ispilolite/queue")
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			carrier := propagation.MapCarrier{}
			for key, value := range delivery.Headers {
				if text, ok := value.(string); ok {
					carrier[key] = text
				}
			}
			messageCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
			messageCtx, span := tracer.Start(messageCtx, "rabbitmq consume", trace.WithSpanKind(trace.SpanKindConsumer))
			var event Event
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				monitoring.QueueConsumed.WithLabelValues(queueName, "invalid").Inc()
				_ = delivery.Nack(false, false)
				span.End()
				continue
			}
			if err := handler(messageCtx, event); err != nil {
				monitoring.QueueConsumed.WithLabelValues(queueName, "error").Inc()
				_ = delivery.Nack(false, false)
				span.End()
				continue
			}
			monitoring.QueueConsumed.WithLabelValues(queueName, "success").Inc()
			_ = delivery.Ack(false)
			span.End()
		}
	}
}

func (s *Service) CollectDepth(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub == nil || s.pub.IsClosed() {
		return
	}
	for _, name := range []string{JobAssignmentQueue, NotificationPushQueue, NotificationSMSQueue} {
		q, err := s.pub.QueueInspect(name)
		if err == nil {
			monitoring.QueueDepth.WithLabelValues(name).Set(float64(q.Messages))
		}
	}
}

func (s *Service) StartDepthCollector(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.CollectDepth(ctx)
			}
		}
	}()
}

// ReplayDLQ republishes up to limit messages from queueName.dlq. A message is
// acknowledged only after the original payload has been accepted by its queue.
func (s *Service) ReplayDLQ(ctx context.Context, queueName string, limit int) (int, error) {
	exchange, key, ok := queueRoute(queueName)
	if !ok {
		return 0, fmt.Errorf("unknown queue %q", queueName)
	}
	if limit <= 0 {
		limit = 100
	}
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || conn.IsClosed() {
		if err := s.reconnect(ctx); err != nil {
			return 0, err
		}
		s.mu.Lock()
		conn = s.conn
		s.mu.Unlock()
	}
	ch, err := conn.Channel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()
	if err := ch.Confirm(false); err != nil {
		return 0, err
	}
	confirmations := ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
	replayed := 0
	for replayed < limit {
		delivery, ok, err := ch.Get(queueName+".dlq", false)
		if err != nil {
			return replayed, err
		}
		if !ok {
			return replayed, nil
		}
		if err := ch.PublishWithContext(ctx, exchange, key, false, false, amqp091.Publishing{DeliveryMode: amqp091.Persistent, ContentType: delivery.ContentType, Headers: delivery.Headers, Body: delivery.Body, MessageId: delivery.MessageId, Type: delivery.Type, Timestamp: delivery.Timestamp}); err != nil {
			return replayed, err
		}
		select {
		case confirmation, ok := <-confirmations:
			if !ok || !confirmation.Ack {
				return replayed, errors.New("rabbitmq rejected replayed message")
			}
		case <-ctx.Done():
			return replayed, ctx.Err()
		}
		if err := delivery.Ack(false); err != nil {
			return replayed, err
		}
		replayed++
	}
	return replayed, nil
}

func queueRoute(queueName string) (string, string, bool) {
	switch queueName {
	case JobAssignmentQueue:
		return JobExchange, "job.replay", true
	case NotificationPushQueue:
		return NotificationExchange, "notification.push", true
	case NotificationSMSQueue:
		return NotificationExchange, "notification.sms", true
	default:
		return "", "", false
	}
}

func (s *Service) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		defaultService.Lock()
		if defaultService.service == s {
			defaultService.service = nil
		}
		defaultService.Unlock()
		s.mu.Lock()
		if s.pub != nil {
			_ = s.pub.Close()
		}
		if s.conn != nil {
			err = s.conn.Close()
		}
		s.mu.Unlock()
	})
	return err
}
