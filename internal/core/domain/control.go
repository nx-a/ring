package domain

import "fmt"

type Control struct {
	ControlId uint64
	Login     string
	Password  string
	Buckets   []Bucket
}

func (c Control) String() string {
	return fmt.Sprintf("Control{ControlId:%d Login:%s Password:%s Buckets:%v}", c.ControlId, c.Login, c.Password, c.Buckets)
}
