import { useLiveGameConnection } from './useLiveGameConnection'

export function ExampleCounter() {
    const { fen, lastMove, turn, svg, isConnected } = useLiveGameConnection()

    console.log({ fen, lastMove, turn, });



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
                <div><strong>Turn:</strong> {turn}</div>
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

