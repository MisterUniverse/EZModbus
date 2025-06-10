// handler.go - Modbus request handler
package handler

import (
	"SPModbus/config"
	"SPModbus/mlog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simonvetter/modbus"
)

type Stats struct {
	RequestsHandled uint64
	Errors          uint64
	StartTime       time.Time
}

type ModbusHandler struct {
	config         config.ModbusConfig
	logger         *mlog.Logger
	mu             sync.RWMutex
	holdingRegs    []uint16
	inputRegs      []uint16
	coils          []bool
	discreteInputs []bool
	counter        uint16
	stats          Stats
}

func NewModbusHandler(config config.ModbusConfig, logger *mlog.Logger) *ModbusHandler {
	h := &ModbusHandler{
		config:         config,
		logger:         logger,
		holdingRegs:    make([]uint16, config.MaxRegisters),
		inputRegs:      make([]uint16, config.MaxRegisters),
		coils:          make([]bool, config.MaxRegisters),
		discreteInputs: make([]bool, config.MaxRegisters),
		stats:          Stats{StartTime: time.Now()},
	}

	for _, data := range config.InitialData {
		if data.Address >= uint16(config.MaxRegisters) {
			logger.Warn("Initial data address out of bounds, skipping", map[string]interface{}{
				"address": data.Address,
				"max":     config.MaxRegisters,
			})
			continue
		}

		switch data.Type {
		case "holding":
			h.holdingRegs[data.Address] = data.Value
		case "input":
			h.inputRegs[data.Address] = data.Value
		case "coil":
			h.coils[data.Address] = (data.Value != 0)
		case "discrete":
			h.discreteInputs[data.Address] = (data.Value != 0)
		default:
			logger.Warn("Unknown initial data type in config, skipping", map[string]interface{}{
				"type": data.Type,
			})
		}
	}

	h.holdingRegs[config.CounterAddress] = 0

	logger.Info("Handler initialized", map[string]interface{}{
		"max_registers": config.MaxRegisters,
		"unit_id":       config.UnitID,
	})

	return h
}

func (h *ModbusHandler) UpdateCounter() {
	h.mu.Lock()
	defer h.mu.Unlock()

	oldValue := h.counter
	h.counter++
	h.holdingRegs[h.config.CounterAddress] = h.counter

	if h.counter == 0 { // Overflow
		h.logger.Warn("Counter overflow, resetting", nil)
		h.counter = 1
		h.holdingRegs[h.config.CounterAddress] = 1
	}

	h.logger.Debug("Counter updated", map[string]interface{}{
		"address": h.config.CounterAddress,
		"old":     oldValue,
		"new":     h.counter,
	})
}

func (h *ModbusHandler) GetStats() Stats {
	return Stats{
		RequestsHandled: atomic.LoadUint64(&h.stats.RequestsHandled),
		Errors:          atomic.LoadUint64(&h.stats.Errors),
		StartTime:       h.stats.StartTime,
	}
}

func (h *ModbusHandler) HandleHoldingRegisters(req *modbus.HoldingRegistersRequest) ([]uint16, error) {
	atomic.AddUint64(&h.stats.RequestsHandled, 1)

	if req.UnitId != h.config.UnitID {
		atomic.AddUint64(&h.stats.Errors, 1)
		h.logger.Warn("Invalid unit ID", map[string]interface{}{
			"requested": req.UnitId,
			"expected":  h.config.UnitID,
		})
		return nil, modbus.ErrIllegalFunction
	}

	if int(req.Addr)+int(req.Quantity) > len(h.holdingRegs) {
		atomic.AddUint64(&h.stats.Errors, 1)
		h.logger.Warn("Address out of bounds", map[string]interface{}{
			"start":    req.Addr,
			"quantity": req.Quantity,
			"max":      len(h.holdingRegs),
		})
		return nil, modbus.ErrIllegalDataAddress
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var res []uint16
	for i := 0; i < int(req.Quantity); i++ {
		addr := int(req.Addr) + i

		if req.IsWrite {
			// Protect counter register
			if uint16(addr) != h.config.CounterAddress {
				old := h.holdingRegs[addr]
				h.holdingRegs[addr] = req.Args[i]
				h.logger.Debug("Register written", map[string]interface{}{
					"address": addr,
					"old":     old,
					"new":     req.Args[i],
				})
			}
		}

		res = append(res, h.holdingRegs[addr])
	}

	operation := "read"
	if req.IsWrite {
		operation = "write"
	}

	h.logger.Debug("Holding registers handled", map[string]interface{}{
		"operation": operation,
		"start":     req.Addr,
		"quantity":  req.Quantity,
	})

	return res, nil
}

func (h *ModbusHandler) HandleInputRegisters(req *modbus.InputRegistersRequest) ([]uint16, error) {
	atomic.AddUint64(&h.stats.RequestsHandled, 1)

	if req.UnitId != h.config.UnitID {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalFunction
	}

	if int(req.Addr)+int(req.Quantity) > len(h.inputRegs) {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalDataAddress
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	var res []uint16
	for i := 0; i < int(req.Quantity); i++ {
		res = append(res, h.inputRegs[int(req.Addr)+i])
	}

	return res, nil
}

func (h *ModbusHandler) HandleCoils(req *modbus.CoilsRequest) ([]bool, error) {
	atomic.AddUint64(&h.stats.RequestsHandled, 1)

	if req.UnitId != h.config.UnitID {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalFunction
	}

	if int(req.Addr)+int(req.Quantity) > len(h.coils) {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalDataAddress
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var res []bool
	for i := 0; i < int(req.Quantity); i++ {
		addr := int(req.Addr) + i

		if req.IsWrite {
			h.coils[addr] = req.Args[i]
		}

		res = append(res, h.coils[addr])
	}

	return res, nil
}

func (h *ModbusHandler) HandleDiscreteInputs(req *modbus.DiscreteInputsRequest) ([]bool, error) {
	atomic.AddUint64(&h.stats.RequestsHandled, 1)

	if req.UnitId != h.config.UnitID {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalFunction
	}

	if int(req.Addr)+int(req.Quantity) > len(h.discreteInputs) {
		atomic.AddUint64(&h.stats.Errors, 1)
		return nil, modbus.ErrIllegalDataAddress
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	var res []bool
	for i := 0; i < int(req.Quantity); i++ {
		res = append(res, h.discreteInputs[int(req.Addr)+i])
	}

	return res, nil
}
