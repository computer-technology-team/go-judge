package storage

import (
	"embed"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/pkg/errors"
)

//go:embed migrations/*.sql
var Migrations embed.FS

func EnsureMigrationsDone(driver database.Driver, dbName string) error {
	srcDriver, err := httpfs.New(http.FS(Migrations), "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"httpfs",
		srcDriver,
		dbName, driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
