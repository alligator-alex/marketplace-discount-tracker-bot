package database

import (
	"bot/internal/app/helpers"
	"context"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Context    context.Context
	Connection *pgxpool.Pool
}

var conn sync.Once
var database *Postgres

func NewPostgres() (*Postgres, error) {
	conn.Do(func() {
		context := context.Background()
		db, err := pgxpool.New(context, getDsn())
		if err != nil {
			log.Fatalln("unable to create connection pool: %w", err)
		}

		database = &Postgres{
			Context:    context,
			Connection: db,
		}

		prepareTables()
	})

	return database, nil
}

func (db *Postgres) Ping() error {
	return db.Connection.Ping(db.Context)
}

func (db *Postgres) CloseConnection() {
	db.Connection.Close()
}

func getDsn() string {
	return helpers.ConcatStrings(
		"postgres://",
		os.Getenv("DB_USERNAME"), ":", url.QueryEscape(os.Getenv("DB_PASSWORD")),
		"@", os.Getenv("DB_HOST"), ":", os.Getenv("DB_PORT"),
		"/", os.Getenv("DB_DATABASE"),
	)
}

func prepareTables() {
	sql := `CREATE TABLE IF NOT EXISTS migrations(
		id SERIAL PRIMARY KEY,
		migration VARCHAR(255) NOT NULL,
		batch INTEGER NOT NULL
	);`

	_, err := database.Connection.Exec(database.Context, sql)

	if err != nil {
		log.Fatalln("Unable to prepare tables:", err)
	}
}
