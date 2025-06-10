// config.go - Configuration management
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Server  ServerConfig  `json:"server"`
	Logging LoggingConfig `json:"logging"`
	Modbus  ModbusConfig  `json:"modbus"`
}

type ServerConfig struct {
	Address    string `json:"address"`
	Port       int    `json:"port"`
	MaxClients uint   `json:"max_clients"`
	Timeout    int    `json:"timeout"`
	MaxRetries int    `json:"max_retries"`
	RetryDelay int    `json:"retry_delay"`
}

type LoggingConfig struct {
	Level   string `json:"level"`
	File    string `json:"file"`
	MaxSize int    `json:"max_size_mb"`
	Console bool   `json:"console"`
}

type RegisterValue struct {
	Type    string `json:"type"`
	Address uint16 `json:"address"`
	Value   uint16 `json:"value"`
}

type ModbusConfig struct {
	UnitID         uint8           `json:"unit_id"`
	MaxRegisters   int             `json:"max_registers"`
	CounterAddress uint16          `json:"counter_address"`
	UpdateInterval int             `json:"update_interval"`
	InitialData    []RegisterValue `json:"initial_data"`
}

func LoadConfig(filename string) (*Config, error) {
	// Default configuration
	config := &Config{
		Server: ServerConfig{
			Address:    "0.0.0.0",
			Port:       1502,
			MaxClients: 10,
			Timeout:    30,
			MaxRetries: 3,
			RetryDelay: 5,
		},
		Logging: LoggingConfig{
			Level:   "INFO",
			File:    "modbus_server.jsonl",
			MaxSize: 100,
			Console: true,
		},
		Modbus: ModbusConfig{
			UnitID:         1,
			MaxRegisters:   1000,
			CounterAddress: 102,
			UpdateInterval: 1,
			InitialData: []RegisterValue{
				{Type: "holding", Address: 100, Value: 2025},
				{Type: "holding", Address: 101, Value: 1234},
				{Type: "coil", Address: 0, Value: 1},
				{Type: "discrete", Address: 0, Value: 1},
				{Type: "input", Address: 100, Value: 5678},
			},
		},
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {

		log.Printf("Config file '%s' not found, creating with defaults", filename)

		if dir := filepath.Dir(filename); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create config directory: %w", err)
			}
		}

		file, err := os.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file '%s': %w", filename, err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(config); err != nil {
			return nil, fmt.Errorf("failed to write config file '%s': %w", filename, err)
		}

		log.Printf("Created config file '%s' - edit it and restart to customize settings", filename)
		return config, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file '%s': %w", filename, err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", filename, err)
	}

	return config, nil
}
