package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// 1. Initialize via Wire
	app, err := InitializeWorker()
	if err != nil {
		logrus.Fatalf("Failed to initialize worker: %v", err)
	}

	log := app.Logger
	log.Info("Worker Service starting...")

	// 2. Ensure Index Mapping Exists
	if err := app.ESRepo.CreateIndexIfNotExists(context.Background()); err != nil {
		log.Fatalf("Failed to ensure index mapping: %v", err)
	}

	// 2. Setup Consumer Topology (Exchange, Queue, Bindings)
	if err := app.Consumer.Setup(); err != nil {
		log.Fatalf("Failed to setup consumer topology: %v", err)
	}

	// 3. Start Consumer in Goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.Consumer.Start(ctx); err != nil {
			log.Errorf("Consumer error: %v", err)
		}
	}()

	log.Info("Worker Service is running. Press CTRL+C to stop.")

	// 4. Wait for Shutdown Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Infof("Received signal: %v. Shutting down...", sig)
	cancel()

	// 5. Cleanup
	if app.MQConn != nil {
		log.Info("Closing RabbitMQ connection...")
		app.MQConn.Close()
	}

	log.Info("Worker Service stopped gracefully.")
}
