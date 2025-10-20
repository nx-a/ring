package migrate

import (
	"embed"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

//go:embed sql/*.sql
var schemaFs embed.FS

type Config interface {
	Get(string) string
}

func Run(cfg Config) error {
	log.Info("run migrate")
	d, err := iofs.New(schemaFs, "sql")
	if err != nil {
		return err
	}
	log.Info(cfg.Get("database.dsn"))
	//	m, err := migrate.NewWithDatabaseInstance("iofs", d, cfg.Get("database.dsn"))
	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.Get("database.dsn"))
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	log.Info("migrate done")
	return nil
}
