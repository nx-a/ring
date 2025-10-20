package dto

type Bucket struct {
	BucketId   uint64 `json:"bucketId"`
	ControlId  uint64 `json:"controlId"`
	SystemName string `json:"systemName"`
	TimeLife   uint   `json:"timeLife"` //Врея жизни в часах
	TimeZone   string `json:"timeZone"`
}
