package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // side-effect: драйвер БД pgx5 для migrate.New (DSN со схемой pgx5://).
	_ "github.com/golang-migrate/migrate/v4/source/file"     // side-effect: источник миграций file:// для migrate.New.
)

// migrateDatabaseURL подставляет схему pgx5:// для golang-migrate с драйвером pgx/v5
// (иначе migrate ищет драйвер postgres в database/sql и падает с «unknown driver postgres»).
func migrateDatabaseURL(dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(dsn, "postgres://")
	case strings.HasPrefix(dsn, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(dsn, "postgresql://")
	default:
		return dsn
	}
}

// MigrateUp применяет SQL-миграции из каталога ./migrations относительно рабочей директории процесса.
func MigrateUp(dsn string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	dir := filepath.Join(wd, "migrations")
	if _, statErr := os.Stat(dir); statErr != nil {
		return fmt.Errorf("migrate: %w", statErr)
	}
	srcURL := "file://" + filepath.ToSlash(dir)
	m, err := migrate.New(srcURL, migrateDatabaseURL(dsn))
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
