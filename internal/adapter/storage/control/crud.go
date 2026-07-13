package control

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Control struct {
	pool          *pgxpool.Pool
	bucketStorage ports.BucketService
}

func New(pool *pgxpool.Pool, bs ports.BucketService) *Control {
	return &Control{pool: pool, bucketStorage: bs}
}
func (c *Control) Registration(control *domain.Control) (domain.Control, error) {
	if control.Role == "" {
		control.Role = "admin"
	}
	var id uint64
	err := c.pool.QueryRow(context.Background(), "insert into control (login, password, role) values ($1, $2, $3) RETURNING control_id", control.Login, hashPassword(control.Password), control.Role).Scan(&id)
	if err != nil {
		return *control, fmt.Errorf("registration failed: %w", err)
	}
	control.ControlId = id
	log.Info(id, control)
	return *control, nil
}
func (c *Control) Auth(login, pass string) (domain.Control, error) {
	var dom domain.Control
	var hashed string
	err := c.pool.QueryRow(context.Background(), "select control_id, login, password, role from control where login = $1", login).Scan(
		&dom.ControlId, &dom.Login, &hashed, &dom.Role,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Control{}, fmt.Errorf("invalid credentials")
		}
		return domain.Control{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pass)); err != nil {
		return domain.Control{}, fmt.Errorf("invalid credentials")
	}
	dom.Buckets, err = c.bucketStorage.GetByControl(dom.ControlId)
	if err != nil {
		log.WithError(err).Warn("failed to load buckets")
		dom.Buckets = nil
	}
	return dom, nil
}
func hashPassword(pass string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("failed to hash password")
		return ""
	}
	return string(hashed)
}
