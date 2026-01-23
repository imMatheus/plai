package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/notnil/chess"
)

type Database struct {
	db *sql.DB
}

type GameRecord struct {
	ID          string    `json:"id"`
	WhitePlayer string    `json:"whitePlayer"`
	BlackPlayer string    `json:"blackPlayer"`
	Winner      *string   `json:"winner"`
	Outcome     string    `json:"outcome"`
	StartedAt   time.Time `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt"`
	TotalMoves  int       `json:"totalMoves"`
	PGN         string    `json:"pgn"`
}

func NewDatabase(databaseURL string) (*Database, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to database")
	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) SaveGame(whitePlayer, blackPlayer string, game *chess.Game, startedAt time.Time) error {
	outcome := game.Outcome()
	outcomeStr := outcome.String()

	var winner *string
	if outcome == chess.WhiteWon {
		w := whitePlayer
		winner = &w
	} else if outcome == chess.BlackWon {
		b := blackPlayer
		winner = &b
	}

	endedAt := time.Now()
	totalMoves := len(game.Moves())
	pgn := game.String()

	query := `
		INSERT INTO games (white_player, black_player, winner, outcome, started_at, ended_at, total_moves, pgn)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := d.db.Exec(query, whitePlayer, blackPlayer, winner, outcomeStr, startedAt, endedAt, totalMoves, pgn)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	log.Printf("Game saved: %s vs %s, outcome: %s, moves: %d", whitePlayer, blackPlayer, outcomeStr, totalMoves)
	return nil
}

func (d *Database) GetAllGames() ([]GameRecord, error) {
	query := `
		SELECT id, white_player, black_player, winner, outcome, started_at, ended_at, total_moves, pgn
		FROM games
		ORDER BY started_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []GameRecord
	for rows.Next() {
		var game GameRecord
		err := rows.Scan(
			&game.ID,
			&game.WhitePlayer,
			&game.BlackPlayer,
			&game.Winner,
			&game.Outcome,
			&game.StartedAt,
			&game.EndedAt,
			&game.TotalMoves,
			&game.PGN,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game row: %w", err)
		}
		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating game rows: %w", err)
	}

	return games, nil
}
