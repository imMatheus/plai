import { useLiveGameConnection } from './useLiveGameConnection'

export function LiveGameBoard() {
    const { fen, lastMove, turn, svg, isConnected, viewerCount, whitePlayer, blackPlayer, currentPlayer } = useLiveGameConnection()

    console.log({ fen, lastMove, turn, svg: svg.length, whitePlayer, blackPlayer, viewerCount });



    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            padding: '2rem',
            fontFamily: 'system-ui, sans-serif'
        }}>
            <div style={{ marginBottom: '1rem' }}>
                <span style={{
                    display: 'inline-block',
                    width: '12px',
                    height: '12px',
                    borderRadius: '50%',
                    backgroundColor: isConnected ? '#4ade80' : '#ef4444',
                    marginRight: '8px'
                }} />
                {isConnected ? 'Connected' : 'Disconnected'}
            </div>

            <div style={{ marginBottom: '1rem', textAlign: 'center' }}>
                <div style={{ fontSize: '1.2rem', fontWeight: 'bold', marginBottom: '0.5rem' }}>
                    {whitePlayer} vs {blackPlayer}
                </div>
                <div><strong>White:</strong> {whitePlayer}</div>
                <div><strong>Black:</strong> {blackPlayer}</div>
                <div><strong>Turn:</strong> {turn} ({currentPlayer})</div>
                <div><strong>Viewer Count:</strong> {viewerCount}</div>
                {lastMove && <div><strong>Last Move:</strong> {lastMove}</div>}
            </div>

            <div
                dangerouslySetInnerHTML={{ __html: svg }}
                style={{
                    maxWidth: '600px',
                    border: '2px solid #333',
                    borderRadius: '8px',
                    overflow: 'hidden'
                }}
            />
        </div>
    )
}

