import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'

interface GameRecord {
  id: string
  whitePlayer: string
  blackPlayer: string
  winner: string | null
  outcome: string
  startedAt: string
  endedAt: string | null
  totalMoves: number
  pgn: string
}

const fetchGames = async (): Promise<GameRecord[]> => {
  const response = await fetch('http://localhost:8080/api/games')
  if (!response.ok) {
    throw new Error('Failed to fetch games')
  }
  const data = await response.json()
  return data || []
}

export function GamesList() {
  const [expandedGame, setExpandedGame] = useState<string | null>(null)

  const { data: games = [], isLoading, error } = useQuery({
    queryKey: ['games'],
    queryFn: fetchGames,
  })

  const toggleGame = (gameId: string) => {
    setExpandedGame(expandedGame === gameId ? null : gameId)
  }

  if (isLoading) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        Loading games...
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center', color: '#ef4444' }}>
        Error: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    )
  }

  return (
    <div style={{
      padding: '2rem',
      maxWidth: '1200px',
      margin: '0 auto'
    }}>
      <h2 style={{
        fontSize: '1.5rem',
        fontWeight: 'bold',
        marginBottom: '1.5rem',
        textAlign: 'center'
      }}>
        Game History ({games.length} games)
      </h2>

      {games.length === 0 ? (
        <div style={{ textAlign: 'center', color: '#666' }}>
          No games recorded yet. Games will appear here after they finish.
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          {games.map((game) => (
            <div
              key={game.id}
              style={{
                border: '1px solid #ddd',
                borderRadius: '8px',
                padding: '1rem',
                backgroundColor: '#fff',
                cursor: 'pointer',
                transition: 'box-shadow 0.2s',
              }}
              onClick={() => toggleGame(game.id)}
              onMouseEnter={(e) => {
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.boxShadow = 'none'
              }}
            >
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '0.5rem'
              }}>
                <div style={{ fontWeight: 'bold', fontSize: '1.1rem' }}>
                  {game.whitePlayer} vs {game.blackPlayer}
                </div>
                <div style={{
                  fontSize: '0.9rem',
                  color: '#666'
                }}>
                  {new Date(game.startedAt).toLocaleString()}
                </div>
              </div>

              <div style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr 1fr',
                gap: '1rem',
                fontSize: '0.9rem'
              }}>
                <div>
                  <strong>Winner:</strong>{' '}
                  {game.winner ? (
                    <span style={{ color: '#16a34a' }}>{game.winner}</span>
                  ) : (
                    <span style={{ color: '#666' }}>Draw</span>
                  )}
                </div>
                <div>
                  <strong>Outcome:</strong> {game.outcome}
                </div>
                <div>
                  <strong>Moves:</strong> {game.totalMoves}
                </div>
              </div>

              {expandedGame === game.id && (
                <div style={{
                  marginTop: '1rem',
                  paddingTop: '1rem',
                  borderTop: '1px solid #eee'
                }}>
                  <div style={{ marginBottom: '0.5rem', fontWeight: 'bold' }}>
                    PGN:
                  </div>
                  <pre style={{
                    backgroundColor: '#f5f5f5',
                    padding: '1rem',
                    borderRadius: '4px',
                    overflow: 'auto',
                    fontSize: '0.85rem',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word'
                  }}>
                    {game.pgn}
                  </pre>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
