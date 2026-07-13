package ring

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type Hook struct {
	client    *Client
	hostname  string
	token     string
	appName   string
	ignoreDir string
	levels    []logrus.Level
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
	// CacheDir — директория файлового кеша логов на случай отсутствия
	// связи с сервером (по умолчанию os.TempDir()/ring-log-cache)
	CacheDir string
	// MaxMemoryCache — лимит кеша логов в памяти в байтах (по умолчанию 20 Мб)
	MaxMemoryCache int64
	// MaxFileCache — лимит файлового кеша логов в байтах (по умолчанию 50 Мб)
	MaxFileCache int64
}

func (p Params) String() string {
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Sprintf("{error: %v}", err)
	}
	return string(b)
}

// NewHook Инициализация logrus hook
//
//lint:ignore U1000 Эта функция используется другими пакетами
//nolint:gocritic // Params передаётся по значению для обратной совместимости API
func NewHook(params Params) (*Hook, error) {
	fmt.Println(params)
	logrus.SetReportCaller(true)
	var opts []Option
	if params.CacheDir != "" {
		opts = append(opts, WithCacheDir(params.CacheDir))
	}
	if params.MaxMemoryCache > 0 {
		opts = append(opts, WithMaxMemoryCache(params.MaxMemoryCache))
	}
	if params.MaxFileCache > 0 {
		opts = append(opts, WithMaxFileCache(params.MaxFileCache))
	}
	_client := NewClient(params.Address, &tls.Config{
		InsecureSkipVerify: true,
	}, opts...)

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	if params.IgnoreDir == "" {
		params.IgnoreDir = findProjectRoot()
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

func findProjectRoot() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return ""
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}
func (h *Hook) Fire(entry *logrus.Entry) error {
	h.sendLog(entry)
	return nil
}
func (h *Hook) sendLog(entry *logrus.Entry) {
	file := ""
	if entry.Caller != nil {
		filename := entry.Caller.File
		if len(entry.Caller.File) > len(h.ignoreDir) {
			filename = filename[len(h.ignoreDir):]
		}
		file = fmt.Sprintf("%s:%d", filename, entry.Caller.Line)
	}
	// Сериализуем в JSON
	data, err := json.Marshal(LogEntry{
		Timestamp: entry.Time.Format(time.RFC3339Nano),
		Level:     entry.Level.String(),
		Message:   entry.Message,
		File:      file,
		Hostname:  h.hostname,
		AppName:   h.appName,
		Token:     h.token,
		Fields:    entry.Data,
	})
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
