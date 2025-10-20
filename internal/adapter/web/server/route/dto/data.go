package dto

import "time"

type Data struct {
	Ext  string    `json:"ext"`
	Time time.Time `json:"time"`
	Data []byte    `json:"data"`
}
