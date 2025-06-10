// server.go - Main server logic
package server

import (
	"SPModbus/config"
	"SPModbus/handler"
	"SPModbus/mlog"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

type ModbusServer struct {
	config  *config.Config
	logger  *mlog.Logger
	handler *handler.ModbusHandler
	server  *modbus.ModbusServer
	wg      sync.WaitGroup
}

func NewModbusServer(config *config.Config, logger *mlog.Logger) *ModbusServer {
	handler := handler.NewModbusHandler(config.Modbus, logger)

	return &ModbusServer{
		config:  config,
		logger:  logger,
		handler: handler,
	}
}

func (s *ModbusServer) Start(ctx context.Context) error {
	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if retryCount > 0 {
			if retryCount >= s.config.Server.MaxRetries {
				return fmt.Errorf("max retries (%d) exceeded", s.config.Server.MaxRetries)
			}

			s.logger.Warn("Retrying server start", map[string]interface{}{
				"attempt": retryCount,
				"max":     s.config.Server.MaxRetries,
			})

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(s.config.Server.RetryDelay) * time.Second):
			}
		}

		if err := s.startServer(ctx); err != nil {
			s.logger.Error("Server start failed", map[string]interface{}{
				"error":   err.Error(),
				"attempt": retryCount + 1,
			})
			retryCount++
			continue
		}

		// If we get here, server started successfully
		return nil
	}
}

func (s *ModbusServer) startServer(ctx context.Context) error {
	// Create modbus server
	address := fmt.Sprintf("tcp://%s:%d", s.config.Server.Address, s.config.Server.Port)

	server, err := modbus.NewServer(&modbus.ServerConfiguration{
		URL:        address,
		Timeout:    time.Duration(s.config.Server.Timeout) * time.Second,
		MaxClients: s.config.Server.MaxClients,
	}, s.handler)

	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	s.server = server

	// Start register updater
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runRegisterUpdater(ctx)
	}()

	// Start health checker
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runHealthChecker(ctx)
	}()

	s.logger.Info("Starting server", map[string]interface{}{
		"address": address,
	})

	// Start server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.logger.Info("Server started successfully", map[string]interface{}{"startup": "server running"})

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

func (s *ModbusServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping server", map[string]interface{}{})

	if s.server != nil {
		s.server.Stop()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All goroutines stopped", map[string]interface{}{})
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout, some goroutines may still be running", map[string]interface{}{})
		return ctx.Err()
	}

	return nil
}

func (s *ModbusServer) runRegisterUpdater(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.config.Modbus.UpdateInterval) * time.Second)
	defer ticker.Stop()

	s.logger.Debug("Register updater started", nil)

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Register updater stopping", nil)
			return
		case <-ticker.C:
			s.handler.UpdateCounter()
		}
	}
}

func (s *ModbusServer) runHealthChecker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := s.handler.GetStats()
			s.logger.Info("Health check", map[string]interface{}{
				"requests_handled": stats.RequestsHandled,
				"errors":           stats.Errors,
				"uptime":           time.Since(stats.StartTime).String(),
			})
		}
	}
}
