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
type DataSelect struct {
	Ext       []string   `json:"ext"`
	Points    []uint64   `json:"points"`
	TimeStart *time.Time `json:"timeStart"`
	TimeEnd   *time.Time `json:"timeEnd"`
	Level     []string   `json:"level"`
	Data      []string   `json:"data"`
	Bucket    string     `json:"bucket"`
}
