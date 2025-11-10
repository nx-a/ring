package backet

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
)

type Control struct {
	pool          *pgxpool.Pool
	bucketStorage ports.BucketService
}

func New(pool *pgxpool.Pool, bs ports.BucketService) *Control {
	return &Control{pool: pool, bucketStorage: bs}
}
func (c *Control) Registration(control domain.Control) (domain.Control, error) {
	var login string
	err := c.pool.QueryRow(context.Background(), "select login from control where login = $1", control.Login).Scan(&login)
	if err != nil {
		return domain.Control{}, err
	}
	if len(login) > 0 {
		return control, fmt.Errorf("user already exists")
	}
	var id uint64
	control.Password = password(control.Password)
	log.Debug(control.Password)
	err = c.pool.QueryRow(context.Background(), "insert into control (login, password) values ($1, $2) RETURNING control_id", control.Login, control.Password).Scan(&id)
	if err != nil {
		log.Error(err)
	}
	control.ControlId = id
	log.Info(id, control)
	return control, nil
}
func (c *Control) Auth(login, pass string) (domain.Control, error) {
	pass = password(pass)
	var dom domain.Control
	err := c.pool.QueryRow(context.Background(), "select control_id, login from control where login = $1 and password = $2", login, pass).Scan(
		&dom.ControlId, &dom.Login,
	)
	if err != nil {
		return domain.Control{}, err
	}
	if dom.ControlId == 0 {
		return dom, fmt.Errorf("no such user")
	}
	dom.Buckets, _ = c.bucketStorage.GetByControl(dom.ControlId)
	return dom, nil
}
func password(pass string) string {
	harsher := sha1.New()
	harsher.Write([]byte(pass))
	return base64.URLEncoding.EncodeToString(harsher.Sum(nil))
}
