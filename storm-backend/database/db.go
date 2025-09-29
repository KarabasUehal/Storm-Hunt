package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

var DB *sql.DB

func InitDB() error {
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	dbname := os.Getenv("MYSQL_DBNAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		user, password, host, port, dbname)

	var err error
	DB, err = sql.Open("mysql", dsn) // Подключаемся к MySQL, для которого импортировали драйвер
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MySQL")
	}

	if err = DB.Ping(); err != nil {
		log.Fatal().Err(err).Msg("MySQL ping failed") // Проверка соединения
	}

	log.Info().Msg("[info] Successfully connected to MySQL")

	createTable := `
    CREATE TABLE IF NOT EXISTS storms (
        id INT AUTO_INCREMENT PRIMARY KEY,
        region VARCHAR(100) NOT NULL,
        latitude FLOAT NOT NULL,
        longitude FLOAT NOT NULL,
        wind_speed INT NOT NULL,
        timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE KEY unique_region_timestamp (region, timestamp)
    );
    `
	_, err = DB.Exec(createTable)
	if err != nil {
		log.Fatal().Err(err).Msg("Error to create table")
		return err
	}

	var count int
	err = DB.QueryRow(`SELECT COUNT(*) FROM storms`).Scan(&count)
	if err != nil {
		log.Error().Err(err).Msg("[error] Failed to count storm-notes")
		return fmt.Errorf("failed to count storm-notes: %w", err)
	}

	insertData := `
    INSERT IGNORE INTO storms (region, latitude, longitude, wind_speed) VALUES
    ('Atlantic', 25.3, -75.2, 140),
    ('Pacific', 15.4, -120.5, 130);
    `

	if count == 0 {
		_, err = DB.Exec(insertData)
		if err != nil {
			log.Fatal().Err(err).Msg("Error to insert data")
			return err
		}
	}

	log.Info().Msg("Database initialized and seeded")

	return nil
}

// Получение глобальной переменной DB
func GetDB() *sql.DB {
	return DB
}

// Закрытие соединения с базой данных
func CloseDB() error {
	if DB == nil { // Проверка, существует ли соединение с БД
		log.Info().Msg("No database connection to close")
		return nil
	}

	if err := DB.Close(); err != nil { // Закрытие соединения с БД
		log.Error().Err(err).Msg("Failed to close database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	log.Info().Msg("Database connection closed")
	DB = nil // Очистка глобальной переменной после закрытия
	return nil
}
