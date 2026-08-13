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
	"ispilolite/pkg/monitoring"
)

const (
	JobExchange="ispilo.job"
	NotificationExchange="ispilo.notification"
	DeadLetterExchange="ispilo.deadletter"
	JobAssignmentQueue="job.assignment"
	NotificationPushQueue="notification.push"
	NotificationSMSQueue="notification.sms"
)

type Config struct{URL string;ConsumerName string;Prefetch int}
type Event struct{ID string `json:"id"`;Type string `json:"type"`;AggregateID string `json:"aggregate_id,omitempty"`;RecipientID string `json:"recipient_id,omitempty"`;Data map[string]interface{} `json:"data,omitempty"`;OccurredAt time.Time `json:"occurred_at"`}
type Handler func(context.Context,Event)error
type Service struct{conn *amqp091.Connection;pub *amqp091.Channel;confirmations <-chan amqp091.Confirmation;mu sync.Mutex;consumerName string;closed chan struct{}}

var defaultService struct{sync.RWMutex;service *Service}
func SetDefault(service *Service){defaultService.Lock();defaultService.service=service;defaultService.Unlock()}
func Publish(ctx context.Context,exchange,key string,event Event)error{defaultService.RLock();service:=defaultService.service;defaultService.RUnlock();if service==nil{return errors.New("rabbitmq is not configured")};return service.Publish(ctx,exchange,key,event)}
func PublishBestEffort(exchange,key string,event Event){ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel();_ = Publish(ctx,exchange,key,event)}

func New(cfg Config)(*Service,error){if cfg.URL==""{return nil,errors.New("rabbitmq url is required")};conn,err:=amqp091.Dial(cfg.URL);if err!=nil{return nil,fmt.Errorf("connect rabbitmq: %w",err)};ch,err:=conn.Channel();if err!=nil{conn.Close();return nil,err};if err=ch.Confirm(false);err!=nil{ch.Close();conn.Close();return nil,err};s:=&Service{conn:conn,pub:ch,confirmations:ch.NotifyPublish(make(chan amqp091.Confirmation,1)),consumerName:cfg.ConsumerName,closed:make(chan struct{})};if err=s.declare(ch);err!=nil{s.Close();return nil,err};return s,nil}
func (s *Service) declare(ch *amqp091.Channel)error{for _,exchange:=range []string{JobExchange,NotificationExchange}{if err:=ch.ExchangeDeclare(exchange,"topic",true,false,false,false,nil);err!=nil{return err}};if err:=ch.ExchangeDeclare(DeadLetterExchange,"topic",true,false,false,false,nil);err!=nil{return err};for _,q:=range []struct{name,exchange,key string}{{JobAssignmentQueue,JobExchange,"job.#"},{NotificationPushQueue,NotificationExchange,"notification.push"},{NotificationSMSQueue,NotificationExchange,"notification.sms"}}{args:=amqp091.Table{"x-dead-letter-exchange":DeadLetterExchange,"x-dead-letter-routing-key":q.name+".dead"};if _,err:=ch.QueueDeclare(q.name,true,false,false,false,args);err!=nil{return err};if err:=ch.QueueBind(q.name,q.key,q.exchange,false,nil);err!=nil{return err};dlq:=q.name+".dlq";if _,err:=ch.QueueDeclare(dlq,true,false,false,false,nil);err!=nil{return err};if err:=ch.QueueBind(dlq,q.name+".dead",DeadLetterExchange,false,nil);err!=nil{return err}};return nil}

func (s *Service) Publish(ctx context.Context,exchange,key string,event Event)error{if event.ID==""{event.ID=uuid.NewString()};if event.OccurredAt.IsZero(){event.OccurredAt=time.Now().UTC()};body,err:=json.Marshal(event);if err!=nil{return err};s.mu.Lock();defer s.mu.Unlock();err=s.pub.PublishWithContext(ctx,exchange,key,false,false,amqp091.Publishing{DeliveryMode:amqp091.Persistent,ContentType:"application/json",MessageId:event.ID,Type:event.Type,Timestamp:event.OccurredAt,Body:body});if err!=nil{monitoring.QueuePublished.WithLabelValues(exchange,key,"error").Inc();return err};select{case confirmation,ok:=<-s.confirmations:if !ok||!confirmation.Ack{monitoring.QueuePublished.WithLabelValues(exchange,key,"nack").Inc();return errors.New("rabbitmq rejected message")};case <-ctx.Done():return ctx.Err()};monitoring.QueuePublished.WithLabelValues(exchange,key,"success").Inc();return nil}

func (s *Service) Consume(ctx context.Context,queue string,prefetch int,handler Handler)error{ch,err:=s.conn.Channel();if err!=nil{return err};if prefetch<=0{prefetch=20};if err=ch.Qos(prefetch,0,false);err!=nil{ch.Close();return err};deliveries,err:=ch.Consume(queue,s.consumerName,false,false,false,false,nil);if err!=nil{ch.Close();return err};go func(){defer ch.Close();for{select{case <-ctx.Done():return;case delivery,ok:=<-deliveries:if !ok{return};var event Event;if err:=json.Unmarshal(delivery.Body,&event);err!=nil{monitoring.QueueConsumed.WithLabelValues(queue,"invalid").Inc();_ = delivery.Nack(false,false);continue};if err:=handler(ctx,event);err!=nil{monitoring.QueueConsumed.WithLabelValues(queue,"error").Inc();_ = delivery.Nack(false,false);continue};monitoring.QueueConsumed.WithLabelValues(queue,"success").Inc();_ = delivery.Ack(false)}}}}();return nil}
func (s *Service) CollectDepth(ctx context.Context){s.mu.Lock();defer s.mu.Unlock();for _,name:=range []string{JobAssignmentQueue,NotificationPushQueue,NotificationSMSQueue}{q,err:=s.pub.QueueInspect(name);if err==nil{monitoring.QueueDepth.WithLabelValues(name).Set(float64(q.Messages))}}}
func (s *Service) StartDepthCollector(ctx context.Context){ticker:=time.NewTicker(15*time.Second);go func(){defer ticker.Stop();for{select{case <-ctx.Done():return;case <-ticker.C:s.CollectDepth(ctx)}}}()}
func (s *Service) Close()error{select{case <-s.closed:return nil;default:close(s.closed)};defaultService.Lock();if defaultService.service==s{defaultService.service=nil};defaultService.Unlock();if s.pub!=nil{_ = s.pub.Close()};if s.conn!=nil{return s.conn.Close()};return nil}
