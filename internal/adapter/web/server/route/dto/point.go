package dto

type Point struct {
	PointId    uint64 `json:"pointId"`
	BucketId   uint64 `json:"bucket"`
	ExternalId string `json:"ext"`
	TimeZone   string `json:"timeZone"`
}
