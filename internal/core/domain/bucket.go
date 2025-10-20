package domain

type Bucket struct {
	BucketId   uint64
	ControlId  uint64
	SystemName string
	TimeLife   uint //Врея жизни в часах
	TimeZone   string
}
