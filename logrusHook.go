package ring

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"sync"
	"time"
)

type Hook struct {
	client    *Client
	hostname  string
	token     string
	appName   string
	ignoreDir string
	levels    []logrus.Level
	mu        sync.Mutex
}
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Hostname  string                 `json:"hostname"`
	AppName   string                 `json:"app_name"`
	Token     string                 `json:"token"`
	File      string                 `json:"file"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}
type Params struct {
	Address   string
	Token     string
	AppName   string
	IgnoreDir string
}

func NewHook(params Params) (*Hook, error) {
	_client := NewClient(params.Address, &tls.Config{
		InsecureSkipVerify: true,
	})

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	hook := &Hook{
		client:    _client,
		hostname:  hostname,
		token:     params.Token,
		appName:   params.AppName,
		ignoreDir: params.IgnoreDir,
		levels:    logrus.AllLevels,
	}

	// Запускаем клиент
	if err := _client.Start(); err != nil {
		return nil, err
	}

	return hook, nil
}
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}
func (h *Hook) Fire(entry *logrus.Entry) error {
	go h.sendLog(entry)
	return nil
}
func (h *Hook) sendLog(entry *logrus.Entry) {
	h.mu.Lock()
	// Создаем структуру лога
	logEntry := LogEntry{
		Timestamp: entry.Time.Format(time.RFC3339),
		Level:     entry.Level.String(),
		Message:   entry.Message,
		File:      entry.Caller.File + ":" + strconv.Itoa(entry.Caller.Line),
		Hostname:  h.hostname,
		AppName:   h.appName,
		Token:     h.token,
		Fields:    make(map[string]interface{}),
	}

	// Добавляем поля
	for key, value := range entry.Data {
		logEntry.Fields[key] = value
	}
	h.mu.Unlock()

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
func (h *Hook) Close() {
	h.client.Stop()
}
