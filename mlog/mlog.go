// logger.go - Structured logging
package mlog

import (
	"SPModbus/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type Logger struct {
	config config.LoggingConfig
	file   *os.File
	mu     sync.Mutex
	level  LogLevel
}

func NewLogger(config config.LoggingConfig) (*Logger, error) {
	var file *os.File
	var err error

	if config.File != "" {
		if dir := filepath.Dir(config.File); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log directory: %w", err)
			}
		}

		file, err = os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	level := INFO
	switch config.Level {
	case "DEBUG":
		level = DEBUG
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	}

	return &Logger{
		config: config,
		file:   file,
		level:  level,
	}, nil
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) log(level LogLevel, levelStr, message string, data map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelStr,
		Message:   message,
		Data:      data,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to file
	if l.file != nil {
		if jsonData, err := json.Marshal(entry); err == nil {
			l.file.Write(jsonData)
			l.file.Write([]byte("\n"))
			l.file.Sync()
		}
	}

	// Write to console
	if l.config.Console {
		dataStr := ""
		if len(data) > 0 {
			if jsonData, err := json.Marshal(data); err == nil {
				dataStr = fmt.Sprintf(" %s", string(jsonData))
			}
		}
		fmt.Printf("[%s] %s: %s%s\n", levelStr, entry.Timestamp.Format("15:04:05"), message, dataStr)
	}
}

func (l *Logger) Debug(message string, data map[string]interface{}) {
	l.log(DEBUG, "DEBUG", message, data)
}

func (l *Logger) Info(message string, data map[string]interface{}) {
	l.log(INFO, "INFO", message, data)
}

func (l *Logger) Warn(message string, data map[string]interface{}) {
	l.log(WARN, "WARN", message, data)
}

func (l *Logger) Error(message string, data map[string]interface{}) {
	l.log(ERROR, "ERROR", message, data)
}
