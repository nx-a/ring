package dto

type Point struct {
	PointId    uint64 `json:"pointId"`
	BacketId   uint64 `json:"backetId"`
	ExternalId string `json:"externalId"`
	TimeZone   string `json:"timeZone"`
}
