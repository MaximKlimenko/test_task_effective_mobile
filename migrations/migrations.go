package migrations

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
)

func RunMigrations(dsn string, logger *logrus.Logger) {
	// Преобразуем DSN в формат для golang-migrate
	migrateDSN := fmt.Sprintf("postgres://%s?sslmode=disable", dsn)

	// Инициализация миграций
	m, err := migrate.New(
		"file://migrations",
		migrateDSN,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize migrations: %v", err)
	}

	// Применение миграций
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		logger.Fatalf("Migration failed: %v", err)
	}

	logger.Info("Migrations applied successfully")
}
