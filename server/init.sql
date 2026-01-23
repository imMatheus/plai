CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    white_player VARCHAR(32) NOT NULL,
    black_player VARCHAR(32) NOT NULL,
    winner VARCHAR(32),
    outcome VARCHAR(32),
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,
    total_moves INTEGER,
    pgn TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_games_started_at ON games(started_at DESC);
