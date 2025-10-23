package hook

import (
	"encoding/json"
	"fmt"
	"github.com/nx-a/ring/client"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

type RingHook struct {
	client    *client.RingClient
	hostname  string
	token     string
	appName   string
	levels    []logrus.Level
	formatter logrus.Formatter
	mu        sync.Mutex
}
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Hostname  string                 `json:"hostname"`
	AppName   string                 `json:"app_name"`
	Token     string                 `json:"token"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewRingHook(address, token, appName string, levels []logrus.Level) (*RingHook, error) {
	_client := client.New(address, 5*time.Second)

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	hook := &RingHook{
		client:    _client,
		hostname:  hostname,
		token:     token,
		appName:   appName,
		levels:    levels,
		formatter: &logrus.JSONFormatter{},
	}

	// Запускаем клиент
	if err := _client.Start(); err != nil {
		return nil, err
	}

	return hook, nil
}
func (h *RingHook) Levels() []logrus.Level {
	return h.levels
}
func (h *RingHook) Fire(entry *logrus.Entry) error {
	go h.sendLog(entry)
	return nil
}
func (h *RingHook) sendLog(entry *logrus.Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Создаем структуру лога
	logEntry := LogEntry{
		Timestamp: entry.Time.Format(time.RFC3339),
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Hostname:  h.hostname,
		AppName:   h.appName,
		Token:     h.token,
		Fields:    make(map[string]interface{}),
	}

	// Добавляем поля
	for key, value := range entry.Data {
		logEntry.Fields[key] = value
	}

	// Сериализуем в JSON
	data, err := json.Marshal(logEntry)
	if err != nil {
		fmt.Printf("Failed to marshal log entry: %v\n", err)
		return
	}

	// Отправляем через клиент
	if err := h.client.Send(data); err != nil {
		fmt.Printf("Failed to send log: %v\n", err)
	}
}
func (h *RingHook) Close() {
	h.client.Stop()
}
