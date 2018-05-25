package logs

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3" // repository assumes sqlite
)

type Repository struct {
	db *sql.DB
}

type Game struct {
	Day              string
	ID               int
	Timestamp        time.Time
	Size             int
	Player1, Player2 string
	Result           string
	Winner           string
	Moves            int
}

func Open(db string) (*Repository, error) {
	sql, err := sql.Open("sqlite3", db)
	if err != nil {
		return nil, err
	}
	return &Repository{db: sql}, nil
}

func (r *Repository) Close() {
	r.db.Close()
}

func (r *Repository) DB() *sql.DB {
	return r.db
}
