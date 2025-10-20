package domain

type Token struct {
	TokenId  uint64
	BucketId uint64
	Bucket   string
	Type     uint8 //1-запись, 2-чтение, 3-полный
	Val      string
}
