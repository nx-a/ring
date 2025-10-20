package domain

import (
	"time"
)

type Data struct {
	DataId  string
	Ext     string
	PointId uint64
	Bucket  string
	Time    time.Time
	Val     []byte
}
