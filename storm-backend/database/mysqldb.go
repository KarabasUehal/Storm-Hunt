package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/source"

	migrate_mysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var GORMDB *gorm.DB
var DB *sql.DB

func InitDB() error {
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD") // Получение переменных окружения
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	dbname := os.Getenv("MYSQL_DBNAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", // Формирование строки подключения к MySQL
		user, password, host, port, dbname)

	var err error
	GORMDB, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,  // DSN
		SkipInitializeWithVersion: true, // Пропуск версии
	}), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Info), // Настройка логов
	})
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Failed to connect to MySQL")
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}

	DB, err = GORMDB.DB()
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Error getting MySQL db from gorm db")
		return fmt.Errorf("failed to get MySQL db from gorm d: %w", err)
	}
	DB.SetMaxIdleConns(10) // Настройка pool'a
	DB.SetMaxOpenConns(100)

	if err = DB.Ping(); err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("MySQL ping failed") // Проверка соединения
		return fmt.Errorf("failed to get ping mysql: %w", err)
	}

	log.Info().Str("component", "mysql").Msg("Successfully connected to MySQL")

	sourceInstance, err := iofs.New(migrationsFS, "migrations") // Source instance для embedded (iofs)
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Error creating iofs source")
		return fmt.Errorf("failed to create iofs source: %w", err)
	}
	defer sourceInstance.Close()

	dbInstance, err := migrate_mysql.WithInstance(DB, &migrate_mysql.Config{}) // Database instance для MySQL
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Error creating mysql instance")
		return fmt.Errorf("failed to create mysql instance: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", // source driver name
		sourceInstance,
		"mysql", // db driver name
		dbInstance,
	)
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Error initializing migrate")
		return fmt.Errorf("failed to init migrate: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations") // Проверка загрузки миграций
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("Cannot read migrations dir")
		return fmt.Errorf("cannot read migrations dir: %w", err)
	}
	log.Info().Str("component", "mysql").Int("files_count", len(entries)).Msg("Migrations files loaded")

	for _, entry := range entries {
		log.Info().Str("component", "mysql").Str("file", entry.Name()).Msg("Found migration file")
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange { // Загрузка миграций с суффиксом "up"
		log.Error().Err(err).Str("component", "mysql").Msg("Error migrating up")
		return fmt.Errorf("failed to migrate up: %w", err)
	}
	if err == migrate.ErrNoChange {
		log.Info().Str("component", "mysql").Msg("No migrations to apply (DB up-to-date)")
	} else {
		log.Info().Str("component", "mysql").Msg("Migrations applied successfully")
	}

	version, dirty, err := m.Version() // Проверка версии миграций
	if err == nil {
		log.Info().Str("component", "mysql").Msgf("Current migration version: %d, dirty: %t", version, dirty)
	} else if err == migrate.ErrNoChange {
		log.Info().Str("component", "mysql").Msg("No migration version (fresh DB)")
	} else {
		log.Warn().Str("component", "mysql").Err(err).Msg("Could not get migration version")
	}

	if err := StormInsertData(GORMDB); err != nil {
		log.Error().Err(err).Msg("Error inserting data")
		return fmt.Errorf("failed to insert data, %w", err)
	}
	log.Info().Str("component", "mysql").Msg("Database initialized and seeded")

	return nil
}

func GetGormDB() *gorm.DB {
	return GORMDB
}

// Получение глобальной переменной DB
func GetDB() (*sql.DB, error) {
	if DB != nil {
		return DB, nil
	}
	return nil, fmt.Errorf("db is nil")
}

// Закрытие соединения с базой данных
func CloseMySQL() error {
	if DB == nil { // Проверка, существует ли соединение с БД
		log.Info().Msg("No database connection to close")
		return nil
	}

	if err := DB.Close(); err != nil { // Закрытие соединения с БД
		log.Error().Err(err).Msg("Failed to close database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	log.Info().Msg("Database connection closed")
	DB = nil // Очистка глобальных переменных после закрытия
	GORMDB = nil
	return nil
}
