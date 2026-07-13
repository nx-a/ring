package control

import (
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
)

type ControlService interface {
	LogIn(login, password string) (domain.Control, error)
	Reg(login, password, role string) (domain.Control, error)
}

type Control struct {
	stor ports.ControlStorage
}

func New(store ports.ControlStorage) *Control {
	return &Control{stor: store}
}

func (c Control) LogIn(login, password string) (domain.Control, error) {
	return c.stor.Auth(login, password)
}

func (c Control) Reg(login, password, role string) (domain.Control, error) {
	if role == "" {
		role = "admin"
	}
	return c.stor.Registration(&domain.Control{
		Login:    login,
		Password: password,
		Role:     role,
	})
}
