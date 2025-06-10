// handler_test.go - Unit tests
package handler

import (
	"SPModbus/config"
	"SPModbus/mlog"
	"testing"

	"github.com/simonvetter/modbus"
)

// TestModbusHandler tests basic holding register operations
func TestModbusHandler(t *testing.T) {
	// Setup: Create a test configuration with enough register space
	// Note: MaxRegisters must be > 101 because NewModbusHandler initializes registers 100 and 101
	cfg := config.ModbusConfig{
		UnitID:         1,
		MaxRegisters:   200, // Need at least 102, using 200 for safety
		CounterAddress: 10,
		UpdateInterval: 1,
	}

	// Create a logger that doesn't output to console during testing
	logger, err := mlog.NewLogger(config.LoggingConfig{
		Level:   "ERROR", // Only log errors during tests
		Console: false,   // Don't clutter test output
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create the handler under test
	handler := NewModbusHandler(cfg, logger)

	// Test 1: Normal holding register read should work
	t.Run("ValidRead", func(t *testing.T) {
		req := &modbus.HoldingRegistersRequest{
			UnitId:   1,     // Valid unit ID
			Addr:     0,     // Start at register 0
			Quantity: 5,     // Read 5 registers
			IsWrite:  false, // Read operation
		}

		res, err := handler.HandleHoldingRegisters(req)
		if err != nil {
			t.Fatalf("Expected no error for valid read, got %v", err)
		}
		if len(res) != 5 {
			t.Fatalf("Expected 5 registers, got %d", len(res))
		}
		t.Logf("Successfully read %d registers", len(res))
	})

	// Test 2: Invalid unit ID should be rejected
	t.Run("InvalidUnitID", func(t *testing.T) {
		req := &modbus.HoldingRegistersRequest{
			UnitId:   99, // Invalid unit ID (we only accept 1)
			Addr:     0,
			Quantity: 1,
			IsWrite:  false,
		}

		_, err := handler.HandleHoldingRegisters(req)
		if err != modbus.ErrIllegalFunction {
			t.Fatalf("Expected ErrIllegalFunction for invalid unit ID, got %v", err)
		}
		t.Log("Correctly rejected invalid unit ID")
	})

	// Test 3: Out of bounds address should be rejected
	t.Run("OutOfBounds", func(t *testing.T) {
		req := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     199, // Near the end of our 200-register space (0-199)
			Quantity: 5,   // This would read registers 199-203, but we only have 0-199
			IsWrite:  false,
		}

		_, err := handler.HandleHoldingRegisters(req)
		if err != modbus.ErrIllegalDataAddress {
			t.Fatalf("Expected ErrIllegalDataAddress for out of bounds, got %v", err)
		}
		t.Log("Correctly rejected out of bounds access")
	})
}

// TestCounterUpdate tests the automatic counter functionality
func TestCounterUpdate(t *testing.T) {
	// Setup: Create config with counter at register 10
	// Note: Need MaxRegisters > 101 for default register initialization
	cfg := config.ModbusConfig{
		UnitID:         1,
		MaxRegisters:   200, // Need at least 102 for registers 100,101 initialization
		CounterAddress: 10,  // Counter will be at register 10
		UpdateInterval: 1,
	}

	// Create logger for testing
	logger, err := mlog.NewLogger(config.LoggingConfig{
		Level:   "ERROR",
		Console: false,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	handler := NewModbusHandler(cfg, logger)

	// Test: Update counter several times and verify it increments
	t.Run("CounterIncrement", func(t *testing.T) {
		// Manually trigger counter updates (normally done by timer)
		for i := 0; i < 5; i++ {
			handler.UpdateCounter()
		}

		// Read the counter register to verify it was updated
		req := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     10, // Read the counter register
			Quantity: 1,  // Just one register
			IsWrite:  false,
		}

		res, err := handler.HandleHoldingRegisters(req)
		if err != nil {
			t.Fatalf("Expected no error reading counter, got %v", err)
		}
		if res[0] != 5 {
			t.Fatalf("Expected counter value 5, got %d", res[0])
		}
		t.Logf("Counter correctly incremented to %d", res[0])
	})

	// Test: Verify counter can't be written to (it's protected)
	t.Run("CounterProtection", func(t *testing.T) {
		// Try to write to the counter register
		req := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     10, // Counter register
			Quantity: 1,
			IsWrite:  true,          // Write operation
			Args:     []uint16{999}, // Try to set it to 999
		}

		res, err := handler.HandleHoldingRegisters(req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// The write should be ignored, value should still be 5 from previous test
		if res[0] == 999 {
			t.Fatalf("Counter register was modified! Expected it to be protected")
		}
		t.Logf("Counter correctly protected from writes, value: %d", res[0])
	})
}

// TestRegisterWrite tests write operations
func TestRegisterWrite(t *testing.T) {
	// Note: Need MaxRegisters > 101 for default register initialization
	cfg := config.ModbusConfig{
		UnitID:         1,
		MaxRegisters:   200, // Need at least 102 for registers 100,101 initialization
		CounterAddress: 10,
		UpdateInterval: 1,
	}

	logger, err := mlog.NewLogger(config.LoggingConfig{
		Level:   "ERROR",
		Console: false,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	handler := NewModbusHandler(cfg, logger)

	// Test: Write and read back a register
	t.Run("WriteAndRead", func(t *testing.T) {
		// Write to register 5
		writeReq := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     5,
			Quantity: 1,
			IsWrite:  true,
			Args:     []uint16{12345}, // Write value 12345
		}

		_, err := handler.HandleHoldingRegisters(writeReq)
		if err != nil {
			t.Fatalf("Failed to write register: %v", err)
		}

		// Read it back
		readReq := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     5,
			Quantity: 1,
			IsWrite:  false,
		}

		res, err := handler.HandleHoldingRegisters(readReq)
		if err != nil {
			t.Fatalf("Failed to read register: %v", err)
		}

		if res[0] != 12345 {
			t.Fatalf("Expected 12345, got %d", res[0])
		}
		t.Logf("Successfully wrote and read back value %d", res[0])
	})
}

// TestInitialValues tests that default registers are set correctly
func TestInitialValues(t *testing.T) {
	cfg := config.ModbusConfig{
		UnitID:         1,
		MaxRegisters:   1000,
		CounterAddress: 102,
		UpdateInterval: 1,
	}

	logger, err := mlog.NewLogger(config.LoggingConfig{
		Level:   "ERROR",
		Console: false,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	handler := NewModbusHandler(cfg, logger)

	// Test: Check initial values are set correctly
	t.Run("DefaultValues", func(t *testing.T) {
		// Read registers 100-102 (should have initial values 2024, 2025, 0)
		req := &modbus.HoldingRegistersRequest{
			UnitId:   1,
			Addr:     100,
			Quantity: 3,
			IsWrite:  false,
		}

		res, err := handler.HandleHoldingRegisters(req)
		if err != nil {
			t.Fatalf("Failed to read initial values: %v", err)
		}

		// Check each expected value
		expected := []uint16{2024, 2025, 0}
		for i, exp := range expected {
			if res[i] != exp {
				t.Fatalf("Register %d: expected %d, got %d", 100+i, exp, res[i])
			}
		}
		t.Logf("Initial values correct: %v", res)
	})
}

// BenchmarkHoldingRegisterRead benchmarks read performance
func BenchmarkHoldingRegisterRead(b *testing.B) {
	// Setup for benchmarking
	cfg := config.ModbusConfig{
		UnitID:         1,
		MaxRegisters:   1000,
		CounterAddress: 10,
		UpdateInterval: 1,
	}

	logger, err := mlog.NewLogger(config.LoggingConfig{
		Level:   "ERROR", // Don't log during benchmarks
		Console: false,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	handler := NewModbusHandler(cfg, logger)

	// Create a typical read request
	req := &modbus.HoldingRegistersRequest{
		UnitId:   1,
		Addr:     0,
		Quantity: 10, // Read 10 registers at once
		IsWrite:  false,
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, err := handler.HandleHoldingRegisters(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
