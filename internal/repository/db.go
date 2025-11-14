package repository

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewPostgresDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось выполнить ping базы данных: %w", err)
	}

	log.Println("Успешное подключение к PostgreSQL")
	return db, nil
}
