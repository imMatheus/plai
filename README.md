# PLAI - AI Chess Battle

Watch Claude and ChatGPT play chess against each other in real-time!

## Setup

### Prerequisites
- Go 1.20+
- Node.js 18+
- Docker & Docker Compose (for local Postgres)

### Environment Variables

Create a `.env` file in the `server/` directory:

```env
OPENAI_API_KEY=your_openai_key_here
ANTHROPIC_API_KEY=your_anthropic_key_here
DATABASE_URL=postgres://plai:plaidev@localhost:5432/plai?sslmode=disable
```

For production, simply update `DATABASE_URL` to point to your hosted Postgres instance:
```env
DATABASE_URL=postgres://user:password@your-host.com:5432/dbname?sslmode=require
```

### Running Locally

1. **Start the Postgres database:**
   ```bash
   docker-compose up -d
   ```

2. **Start the Go backend:**
   ```bash
   cd server
   go run .
   ```

3. **Start the frontend (in a new terminal):**
   ```bash
   cd web
   npm install
   npm run dev
   ```

4. **Visit http://localhost:5173** to watch the game!

### Database Management

- **Start database:** `docker-compose up -d`
- **Stop database:** `docker-compose down`
- **View logs:** `docker-compose logs -f postgres`
- **Reset database:** `docker-compose down -v` (⚠️ deletes all data)

### Database Schema

Games are automatically saved to the database when they finish with the following schema:

```sql
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    white_player TEXT NOT NULL,
    black_player TEXT NOT NULL,
    winner TEXT,
    outcome TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,
    total_moves INTEGER,
    pgn TEXT NOT NULL
);
```

## API Endpoints

- `GET /api/games` - Returns all games in JSON format, sorted by most recent first
- `WS /ws` - WebSocket connection for live game updates

## Features

- ♟️ Real-time chess games between Claude and ChatGPT
- 📊 Complete game history with PGN notation
- 👥 Live viewer count
- 🎨 Beautiful SVG board rendering
- 💾 Persistent game storage in Postgres
