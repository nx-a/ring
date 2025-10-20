package ports

import "github.com/nx-a/ring/internal/core/domain"

type ControlService interface {
	LogIn(login, password string) (domain.Control, error)
	Reg(login string, password string) (domain.Control, error)
}
