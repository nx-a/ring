package domain

type Bucket struct {
	BucketId   uint64
	ControlId  uint64
	SystemName string
	TimeLife   uint // время жизни в часах
	TimeZone   string
}
