package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ispilolite/pkg/monitoring"
	"ispilolite/pkg/queue"
	"ispilolite/pkg/telemetry"
)

func main() {
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "ispilolite-notification")
	if err != nil {
		log.Fatalf("telemetry startup failed: %v", err)
	}
	defer shutdownTelemetry(context.Background())
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}
	webhook := os.Getenv("NOTIFICATION_WEBHOOK_URL")
	if webhook == "" {
		log.Fatal("NOTIFICATION_WEBHOOK_URL is required")
	}
	service, err := queue.New(queue.Config{URL: rabbitURL, ConsumerName: getenv("RABBITMQ_CONSUMER_NAME", "notification-worker")})
	if err != nil {
		log.Fatalf("rabbitmq startup failed: %v", err)
	}
	defer service.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &http.Client{Timeout: 10 * time.Second}
	handler := func(ctx context.Context, event queue.Event) error { return deliver(ctx, client, webhook, event) }
	prefetch := getenvInt("RABBITMQ_PREFETCH", 20)
	if err := service.Consume(ctx, queue.NotificationPushQueue, prefetch, handler); err != nil {
		log.Fatalf("consume push queue: %v", err)
	}
	if err := service.Consume(ctx, queue.NotificationSMSQueue, prefetch, handler); err != nil {
		log.Fatalf("consume sms queue: %v", err)
	}
	service.StartDepthCollector(ctx)
	mux := http.NewServeMux()
	mux.Handle("/metrics", monitoring.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	server := &http.Server{Addr: getenv("MONITORING_ADDR", ":9091"), Handler: monitoring.Middleware("notification-worker", mux), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("monitoring server failed: %v", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

func deliver(ctx context.Context, client *http.Client, url string, event queue.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", event.ID)
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("notification provider status %d", response.StatusCode)
	}
	return nil
}
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func getenvInt(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil && value > 0 {
		return value
	}
	return fallback
}
