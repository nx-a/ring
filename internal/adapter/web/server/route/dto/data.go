package dto

import (
	"encoding/json"
	"time"
)

type Data struct {
	Ext   string          `json:"ext"`
	Time  *time.Time      `json:"time"`
	Level string          `json:"level"`
	Data  json.RawMessage `json:"data"`
}
