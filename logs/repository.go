package logs

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // repository assumes sqlite
)

type Repository struct {
	db *sql.DB

	insert *sql.Stmt
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
	_, err = sql.Exec(createGameTable)
	if err != nil {
		sql.Close()
		return nil, fmt.Errorf("create game table: %v", err)
	}
	_, err = sql.Exec(createPlayerTable)
	if err != nil {
		sql.Close()
		return nil, fmt.Errorf("create player_game table: %v", err)
	}

	repo := &Repository{db: sql}
	repo.insert, err = sql.Prepare(insertStmt)
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("prepare: %v", err)
	}
	return repo, nil
}

func (r *Repository) InsertGame(g *Game) error {
	return r.insertGame(r.insert, g)
}

func (r *Repository) insertGame(stmt *sql.Stmt, g *Game) error {
	_, err := stmt.Exec(
		g.Day, g.ID, g.Timestamp,
		g.Size, g.Player1, g.Player2,
		g.Result, g.Winner, g.Moves,
	)
	return err
}

func (r *Repository) InsertGames(gs []*Game) error {
	txn, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	stmt := txn.Stmt(r.insert)
	for _, g := range gs {
		if e := r.insertGame(stmt, g); e != nil {
			return e
		}
	}
	return txn.Commit()
}

func (r *Repository) Close() {
	r.db.Close()
}
