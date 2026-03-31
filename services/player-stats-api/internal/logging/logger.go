package logging

import (
	"context"
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

type requestFieldsKey struct{}

type RequestFields struct {
	mu     sync.Mutex
	fields map[string]any
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

func WithRequestFields(ctx context.Context) (context.Context, *RequestFields) {
	fields := &RequestFields{
		fields: make(map[string]any),
	}

	return context.WithValue(ctx, requestFieldsKey{}, fields), fields
}

func RequestFieldsFromContext(ctx context.Context) *RequestFields {
	fields, _ := ctx.Value(requestFieldsKey{}).(*RequestFields)
	return fields
}

func (f *RequestFields) Add(values map[string]any) {
	if f == nil || len(values) == 0 {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for key, value := range values {
		f.fields[key] = value
	}
}

func (f *RequestFields) Snapshot() map[string]any {
	if f == nil {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.fields) == 0 {
		return nil
	}

	snapshot := make(map[string]any, len(f.fields))
	for key, value := range f.fields {
		snapshot[key] = value
	}

	return snapshot
}
