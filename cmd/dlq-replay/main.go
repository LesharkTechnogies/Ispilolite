package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ispilolite/pkg/queue"
)

func main() {
	queueName := flag.String("queue", "", "source queue name")
	limit := flag.Int("limit", 100, "maximum messages to replay")
	flag.Parse()
	if *queueName == "" {
		log.Fatal("-queue is required")
	}
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	service, err := queue.New(queue.Config{URL: rabbitURL, ConsumerName: "dlq-replay"})
	if err != nil {
		log.Fatalf("rabbitmq startup failed: %v", err)
	}
	defer service.Close()

	replayed, err := service.ReplayDLQ(ctx, *queueName, *limit)
	if err != nil {
		log.Fatalf("replay failed after %d messages: %v", replayed, err)
	}
	fmt.Printf("replayed %d messages from %s.dlq\n", replayed, *queueName)
}
