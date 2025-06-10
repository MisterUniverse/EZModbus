// main.go
package main

import (
	"SPModbus/config"
	"SPModbus/mlog"
	"SPModbus/server"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var configFile = flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// Create logger
	logger, err := mlog.NewLogger(config.Logging)
	if err != nil {
		log.Println(config.Logging)
		log.Fatalf("Failed to create logger: %v\n", err)
	}
	defer logger.Close()

	logger.Info("Starting Modbus server", map[string]interface{}{
		"version": "1.0.0",
		"config":  *configFile,
	})

	// Create and start srvr
	srvr := server.NewModbusServer(config, logger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	if err := srvr.Start(ctx); err != nil {
		logger.Error("Failed to start server", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received", map[string]interface{}{"shutdown": "Shutting down"})

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srvr.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Server stopped successfully", map[string]interface{}{"shutdown": "End"})
}
