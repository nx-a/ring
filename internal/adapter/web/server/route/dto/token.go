package dto

type Token struct {
	TokenId  uint64 `json:"tokenId"`
	BucketId uint64 `json:"bucket"`
	Type     uint8  `json:"type"` //1-запись, 2-чтение, 3-полный
	Val      string `json:"token"`
}
