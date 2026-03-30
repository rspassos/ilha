package logging

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type Logger struct {
	mu      sync.Mutex
	writer  io.Writer
	now     func() time.Time
	service string
}

func New(writer io.Writer, service string) *Logger {
	return &Logger{
		writer:  writer,
		now:     time.Now().UTC,
		service: service,
	}
}

func (l *Logger) Info(message string, fields map[string]any) error {
	return l.log("info", message, fields)
}

func (l *Logger) Warn(message string, fields map[string]any) error {
	return l.log("warn", message, fields)
}

func (l *Logger) Error(message string, fields map[string]any) error {
	return l.log("error", message, fields)
}

func (l *Logger) log(level string, message string, fields map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record := map[string]any{
		"ts":      l.now().Format(time.RFC3339),
		"level":   level,
		"service": l.service,
		"message": message,
	}
	for key, value := range fields {
		record[key] = value
	}

	return json.NewEncoder(l.writer).Encode(record)
}
